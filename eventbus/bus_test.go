package eventbus

import "testing"
import "fmt"
import "time"

func TestBusSimple(t *testing.T) {
	bus := NewEventBus()
	m := "Hello, World!"
	x := ""

	// Register the handler.
	bus.RegisterHandler("foo", func(e Event) EventHandleResult {
		println("normal event handler invoked")
		e2 := e.(FooEvent)
		x = e2.msg
		return EHANDLE_OK
	})
	if bus.CountHandlers("foo") != 1 {
		t.Fail()
	}

	// Publish an event to the handler.
	bus.Publish(FooEvent{
		msg:   m,
		async: false,
	})
	if x != m {
		t.Fail()
	}
}

func TestBusAsync(t *testing.T) {

	bus := NewEventBus()
	c := make(chan uint8, 2)

	// Register the handler.
	bus.RegisterHandler("foo", func(e Event) EventHandleResult {
		println("async event handler invoked")
		c <- 42
		return EHANDLE_OK
	})

	// Escape if we don't work out.
	go (func() {
		time.Sleep(1000 * time.Millisecond)
		t.FailNow()
	})()

	// Publish an event to the handler.
	bus.Publish(FooEvent{
		msg:   "asdf",
		async: true,
	})

	r := <-c
	fmt.Printf("got result: %d\n", r)
	return

}

type FooEvent struct {
	msg   string
	async bool
}

func (FooEvent) Name() string {
	return "foo"
}

func (e FooEvent) Flags() uint8 {
	if e.async {
		return EFLAG_ASYNC
	} else {
		return EFLAG_NORMAL
	}
}
