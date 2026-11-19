// Package tll is a thin cgo binding to libtll (https://github.com/shramov/tll).
//
// The binding does not hide the C object model: every object that libtll creates for you is
// released explicitly, and the Go garbage collector knows nothing about C memory. The rules
// below are the ones that matter in practice; README.md has the long version with examples.
//
// Ownership:
//
//   - Config, Scheme, Loop, Context, Channel and CallbackHandle are released with Free.
//   - Channel.Close only closes a channel. It does not release it: a closed channel still
//     holds a reference to its Context, so a Context stays alive (with its registries,
//     schemes and buffers) until every Channel created from it is released with Channel.Free.
//     Context.Free only drops the caller's own reference.
//
// Lifecycle of a channel:
//
//   - Create a channel once, then Open and Close it as many times as needed. Re-creating a
//     channel (and its Context) per reconnect attempt is not how libtll is meant to be used.
//   - A channel that is in StateError can only be closed. Close it before opening it again.
//   - Channel.Close may leave a channel in StateClosing (long close) and further non-forced
//     Close calls are ignored; Channel.Open then fails with "invalid state Closing".
//     Channel.CloseForce(true) closes immediately, never fails and is valid in any state.
//
// Processing:
//
//   - Channels make progress only while a Loop is stepped (Loop.Step). Callbacks run inside
//     Step, on the goroutine that calls it, and the Message passed to a callback is valid only
//     until the callback returns (use Message.Copy to keep it).
//   - libtll does not promise that a channel can be used from several threads. Keep Open,
//     Close, Post and Step for one channel on one goroutine.
package tll
