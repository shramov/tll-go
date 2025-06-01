package tll

// #cgo pkg-config: tll
// #include <tll/channel.h>
import "C"

type State int

const (
	StateClosed  State = C.TLL_STATE_CLOSED
	StateOpening State = C.TLL_STATE_OPENING
	StateActive  State = C.TLL_STATE_ACTIVE
	StateClosing State = C.TLL_STATE_CLOSING
	StateError   State = C.TLL_STATE_ERROR
	StateDestroy State = C.TLL_STATE_DESTROY
)
