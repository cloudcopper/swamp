package infra

import (
	"sync"

	"github.com/cloudcopper/swamp/ports"
	"github.com/cskr/pubsub/v2"
)

type EventBus struct {
	bus *pubsub.PubSub[ports.Topic, ports.Event]
	chs map[chan ports.Event]chan ports.Event
}

func NewEventBus() *EventBus {
	bus := &EventBus{
		bus: pubsub.New[ports.Topic, ports.Event](1),
		chs: make(map[chan ports.Event]chan ports.Event),
	}
	return bus
}

func (e *EventBus) Shutdown() {
	e.bus.Shutdown()
}

func (e *EventBus) Pub(topic ports.Topic, event ports.Event) {
	e.bus.Pub(event, topic)
}
func (e *EventBus) Unsub(ch chan ports.Event) {
	inp := e.chs[ch]
	delete(e.chs, ch)
	e.bus.Unsub(inp)
}

// Sub returns elastic channel subscribed to topic.
// The cskr/pubsub/v2 might suffer deadlocks,
// when subscribers are not reading out events fast enough.
func (e *EventBus) Sub(topic ports.Topic) chan ports.Event {
	inp := e.bus.Sub(topic)
	out := make(chan ports.Event, 1)
	e.chs[out] = inp

	mutex := &sync.Mutex{}
	cond := sync.NewCond(mutex)
	fifo := []ports.Event{}
	abort := false

	// Push events to fifo and signal
	go func() {
		for event := range inp {
			cond.L.Lock()
			fifo = append(fifo, event)
			cond.Signal()
			cond.L.Unlock()
		}

		cond.L.Lock()
		abort = true
		cond.Signal()
		cond.L.Unlock()
	}()
	// By signal read elements from fifo and pass forward
	go func() {
		cond.L.Lock()
		defer cond.L.Unlock()
		defer close(out)
		for {
			for len(fifo) > 0 {
				e := fifo[0]
				fifo = fifo[1:]
				cond.L.Unlock()
				out <- e
				cond.L.Lock()
			}
			if abort {
				return
			}
			cond.Wait()
		}
	}()

	return out
}
