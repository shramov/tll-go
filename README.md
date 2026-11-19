# tll-go

Go bindings for [libtll](https://github.com/shramov/tll) (cgo, `pkg-config: tll`).

This guide is about **using** channels from Go: who owns what, when memory is released, and how
to reconnect without leaking. Implementing channels in Go (`tll.Base`, `tll.Prefix`, see
`examples/echo`) is not covered here. For libtll itself read its documentation
(`doc/channel.rst`, `doc/processor.rst`); the lifecycle rules below follow it.

The binding is thin. It does not hide the C object model and the Go garbage collector knows
nothing about C memory, so everything libtll allocates for you is released explicitly.

## Objects and who releases them

| Object | Created by | Released by | Notes |
|---|---|---|---|
| `Config` | `NewConfig`, `LoadConfig` | `Free` | Reference counted. Objects that keep a config hold their own reference. |
| `Scheme` | `NewScheme` | `Free` | Reference counted. A scheme returned by `Channel.Scheme` belongs to the channel; do not free it. |
| `Context` | `NewContext`, `NewContextCfg` | `Free` | Reference counted, see below. |
| `Channel` | `Context.Channel` | **`Free`** | `Close` does **not** release it. |
| `Loop` | `NewLoop` | `Free` | |
| `CallbackHandle` | `Channel.CallbackAdd` | `Free` | Released automatically only when the channel is destroyed. |

Two rules cause most of the trouble:

1. **`Close` is not `Free`.** `Close` ends a connection and keeps the channel object, so that it
   can be opened again. `Free` destroys the object.
2. **A channel keeps its context alive.** Every channel holds a reference to the `Context` it
   was created from, and gives it back only in `Channel.Free`. `Context.Free` drops just the
   caller's reference. A closed channel that was never freed therefore keeps its whole context
   (registries, loaded schemes, buffers) in memory forever.

## Channel states

`Closed` → `Opening` → `Active` → `Closing` → `Closed`, and `Error` from `Opening`, `Active` or
`Closing`. From the libtll documentation:

- a channel in `Error` accepts nothing but `Close`; close it before opening it again;
- `Close` may be long (`Closing`), and while the channel is `Closing` further non-forced `Close`
  calls are ignored;
- forced close (`CloseForce(true)`) puts the channel into `Closed` immediately, cannot fail and
  is allowed in any state.

`Open` requires `Closed`. Opening a channel that is still `Closing` fails with
`invalid state Closing`. This happens with channels that have a long close (prefix and composite
channels, for example `stream+pub+tcp://...`); it was observed after a failed `Open` of such a
server channel (port already in use) followed by a plain `Close`. Use `CloseForce(true)` before
reopening.

`Open` returning `nil` does not mean `Active`: many channels (`tcp`, `stream`, `resolve`)
reach `Active` only while the loop is stepped.

## Reconnecting: create once, open and close

libtll is designed around creating channels once and reopening them. A `Loop` follows state
changes of the channels in it, so a channel added once keeps working after `Close` and `Open`.
Prefix channels such as `resolve://service/channel` look the address up again on every `Open`
when `resolve.mode=always` is set, so a reconnect also picks up a moved service.

Do not create a `Context`, `Loop` and channel for every attempt. Besides the cost, any missing
`Free` then leaks a whole context per attempt.

```go
// run keeps one channel connected until ctx is done.
func run(ctx context.Context, url string) error {
	cfg := tll.NewConfig()
	defer cfg.Free()
	cfg.Set("resolve.mode", "always")
	cfg.Set("resolve.request", "tcp://127.0.0.1:5555;mode=client")

	tllCtx := tll.NewContextCfg(cfg) // takes its own reference to cfg
	defer tllCtx.Free()
	loop := tll.NewLoop()
	defer loop.Free()
	ch := tllCtx.Channel(url) // for example "resolve://service/channel"
	if ch == nil {
		return errors.New("cannot create channel")
	}
	// Runs before loop.Free and tllCtx.Free. A Loop forgets a destroyed channel by itself.
	defer ch.Free()
	if rc := loop.Add(*ch); rc != 0 {
		return fmt.Errorf("loop.Add: %d", rc)
	}

	const first = time.Second
	pause := first
	for ctx.Err() == nil {
		if err := ch.Open(); err != nil {
			log.Printf("open: %v", err)
		} else {
			for ctx.Err() == nil && ch.State() != tll.StateError && ch.State() != tll.StateClosed {
				// Callbacks run in here. libtll currently always returns 0 from step; the check
				// is there so that an error reported in the future is not ignored.
				if rc := loop.Step(100 * time.Millisecond); rc != 0 {
					log.Printf("step: %d", rc)
					break
				}
				if ch.State() == tll.StateActive {
					pause = first // it worked: after a drop the back-off starts over
				}
			}
		}
		_ = ch.CloseForce(true) // allowed in any state, leaves the channel Closed
		select {
		case <-ctx.Done():
		case <-time.After(pause):
		}
		pause = min(pause*2, time.Minute) // back off between attempts (min: Go 1.21 builtin)
	}
	return ctx.Err()
}
```

The snippet uses the `min` builtin (Go 1.21; this module requires Go 1.22), replace it when
copying into an older project. A connection that reaches `Active` and drops at once resets the
back-off every time; if that matters, reset it only after the channel stayed active for a while.

`examples/reconnect` measures what each way of reconnecting costs; it reads `/proc/self/status`
and therefore needs Linux.

## Callbacks and messages

- `Channel.CallbackAdd(cb, mask)` registers a callback and returns a `*CallbackHandle`.
  Callbacks run **inside `Loop.Step`**, on the goroutine that calls `Step`.
- The `Message` passed to a callback points into libtll memory and is valid only until the
  callback returns. Use `Message.Copy()` to keep it.
- Remove a callback with `CallbackHandle.Free()` when you no longer need it. It is released
  automatically only when the channel is destroyed. A channel that you reopen is not, so
  callbacks registered on every attempt would pile up. `Free` may be called from inside a
  callback (libtll does that itself), but not from another goroutine while the loop is stepped.
- `Channel.ChanAddBuffered` is a shortcut that delivers copies into a Go channel, with
  caveats: it blocks the whole `Step` while the buffer is full and nobody reads (until its
  context is done); the returned Go channel is closed only when a message arrives after the
  context is done; and it does not return the handle, so it cannot be removed from a channel
  you reuse. For long-lived channels register your own callback with `CallbackAdd`.

## Threads

libtll does not promise that a channel can be used from several threads. Keep `Open`, `Close`,
`Post` and `Loop.Step` for a channel on one goroutine, and send work to that goroutine through a
Go channel. Before `Channel.Free` or `Loop.Free`, stop the goroutine that is inside `Step`:
freeing under it is a use after free.

## Common mistakes

- **`Close` without `Free`.** Leaks the channel and its context. Reproduce it with
  `go run ./examples/reconnect -mode recreate -free=false`: on libtll 0.6.4 (Linux, x86_64) the
  process grew by about 0.5 MB per attempt, 1558 MB after 3000 attempts, while `-free=true` and
  `-mode reuse` stayed at +3 MB. With retries every couple of seconds that is gigabytes within
  hours. The numbers are from one machine; the point is the trend.
- **`Open` right after `Close`.** Can fail with `invalid state Closing` on channels with a long close; use `CloseForce(true)`.
- **Reopening a channel in `Error`.** Close it first.
- **Using a `Channel` after `Free`.** Methods take `Channel` by value, so every copy keeps the
  same dangling pointer.
- **`Context.Free` while channels exist.** It only drops your reference; the context lives on
  until the last channel is freed.
- **Forgetting to free the config passed to `NewContextCfg`.** The context takes its own
  reference, so the caller still has to release `cfg`.

## Where the rules come from

These are not guesses about the binding; they follow from libtll. Checked against libtll
v0.6.4-8 (`2f71706`); paths are relative to its repository:

- A channel references its context: `tll_channel_new` takes a reference (`tll_channel_context_ref`)
  and `tll_channel_free` gives it back, in `src/channel/context.cc`. `tll_channel_context_free`
  only drops one reference.
- `tll_channel_free` force-closes a channel that is not closed, removes it from the context and
  destroys it (same file).
- States, `Error` handling, long close and forced close: `doc/channel.rst`, sections *State*,
  *Opening* and *Closing*.
- A loop subscribes to a channel's state when it is added, follows `Opening`, `Closed` and
  `Error`, and removes the channel on `Destroy`; its destructor detaches the channels still in
  it: `tll_processor_loop_t` in `src/tll/processor/loop.h`.
- Callbacks may be removed from callbacks: `_state_callback` in `src/channel/impl.c` and the loop
  above both call `tll_channel_callback_del` while handling a state message.
- Re-resolving on every `Open` with `resolve.mode=always`: `src/channel/resolve.rst` and
  `src/channel/resolve.cc`.
