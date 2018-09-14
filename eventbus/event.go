package eventbus

// An Event is a description of "something" that has taken place.
type Event interface {
	Name() string
	Flags() uint8
}

const (

	// EFLAG_NORMAL means this is a normal sync event.
	EFLAG_NORMAL = 0

	// EFLAG_UNCANCELLABLE means that the event cannot be cancelled.
	EFLAG_UNCANCELLABLE = 1 << 0

	// EFLAG_ASYNC_UNSAFE means that the event is an async event.  Don't use declaritvely.
	EFLAG_ASYNC_UNSAFE = 1 << 1 // Don't use this directly!

	// EFLAG_ASYNC means thtat the event will be processed asychronously.
	EFLAG_ASYNC = EFLAG_ASYNC_UNSAFE | EFLAG_UNCANCELLABLE
)
