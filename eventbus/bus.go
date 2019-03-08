package eventbus

import (
	"fmt"
	"sync"

	"github.com/mit-dci/lit/logging"
)

// An EventBus takes events and forwards them to event handlers matched by name.
type EventBus struct {
	handlers     map[string][]*eventhandler
	eventMutexes map[string]*sync.Mutex
	mutex        *sync.Mutex
}

// NewEventBus creates a new event bus without any event handlers.
func NewEventBus() EventBus {
	return EventBus{
		handlers:     map[string][]*eventhandler{},
		eventMutexes: map[string]*sync.Mutex{}, // Make triggering events safe.
		mutex:        &sync.Mutex{},            // Make adding events/handlers safe.
	}
}

const (

	// EHANDLE_OK means that the event should not be cancelled.
	EHANDLE_OK = 0

	// EHANDLE_CANCEL means that the event should be cancelled.
	EHANDLE_CANCEL = 1
)

// EventHandleResult is a flag field to represent certain things.
type EventHandleResult uint8

type eventhandler struct {
	handleFunc func(Event) EventHandleResult
	mutex      *sync.Mutex // Make sure we don't race a handler against itself.
}

// RegisterHandler registers an event handler function by name
func (b *EventBus) RegisterHandler(eventName string, hFunc func(Event) EventHandleResult) {

	b.mutex.Lock()

	h := &eventhandler{
		handleFunc: hFunc,
		mutex:      &sync.Mutex{},
	}

	// We might need to make a new array of handlers.
	if _, ok := b.handlers[eventName]; !ok {
		b.handlers[eventName] = make([]*eventhandler, 0)
		b.eventMutexes[eventName] = &sync.Mutex{}
	}

	b.handlers[eventName] = append(b.handlers[eventName], h)
	logging.Infof("Registered handler for %s\n", eventName)

	b.mutex.Unlock()

}

// CountHandlers is a convenience function.
func (b *EventBus) CountHandlers(name string) int {
	if _, ok := b.handlers[name]; !ok {
		return 0
	}
	return len(b.handlers[name])
}

// Publish sends an event to the relevant event handlers.
func (b *EventBus) Publish(event Event) (bool, error) {

	logging.Debugf("eventbus: Published event: %s\n", event.Name())

	ck := checkEventSanity(event)
	if ck != nil {
		return true, ck
	}

	name := event.Name()

	// Make a copy of the handler list so we don't block for longer than we need to.
	b.mutex.Lock()
	eventMutex, present := b.eventMutexes[name]
	if !present {
		b.mutex.Unlock() // unlock it early
		return true, nil
	}
	eventMutex.Lock()
	src := b.handlers[name]
	hs := make([]*eventhandler, len(src))
	copy(hs, src)
	b.mutex.Unlock()

	// Figure out the flags.
	f := event.Flags()
	async := (f & EFLAG_ASYNC_UNSAFE) != 0
	uncan := (f & EFLAG_UNCANCELLABLE) != 0

	// Actually iterate over all the handlers and make them run.
	ok := true
	for _, h := range hs {

		if async {

			// If it's an async event, spawn a goroutine for it.  Ignore results.
			go callEventHandler(h, event)

		} else {

			// Since it's not async we might cancel it.
			res, err := callEventHandler(h, event)
			if err != nil {
				logging.Warnf("Error in event handler for %s: %s", name, err.Error())
			}

			if res == EHANDLE_CANCEL && !uncan {
				ok = false
			}

		}

	}

	eventMutex.Unlock()
	return ok, nil

}

// PublishNonblocking sends async events off to the relevant handlers witout blocking.
func (b *EventBus) PublishNonblocking(event Event) error {

	// Make sure it's async, if it is then we can't do it nonblockingly.
	async := (event.Flags() & EFLAG_ASYNC_UNSAFE) != 0
	if !async {
		return fmt.Errorf("event %s not async but called on function that needs async", event.Name())
	}

	// This is the lazy way of spawning it.
	go b.Publish(event)
	return nil

}

func callEventHandler(handler *eventhandler, event Event) (EventHandleResult, error) {
	handler.mutex.Lock()
	r := handler.handleFunc(event)
	handler.mutex.Unlock()
	return r, nil // TODO Catch panics.
}

func checkEventSanity(e Event) error {
	f := e.Flags()

	// If it's async then the caller will return before the event handler can know if it wants to cancel the event.
	if (f&EFLAG_ASYNC_UNSAFE) != 0 && (f&EFLAG_UNCANCELLABLE) == 0 {
		return fmt.Errorf("event of type %s flagged as async but isn't cancellable, is it using EFLAG_ASYNC_UNSAFE instead of EFLAG_ASYNC?", e.Name())
	}
	return nil
}
