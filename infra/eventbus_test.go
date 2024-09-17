package infra

import (
	"sync"
	"testing"

	"github.com/cloudcopper/swamp/ports"
)

func TestEventBus(t *testing.T) {
	bus := NewEventBus()
	ch := bus.Sub("topic1")
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for e := range ch {
			_ = e
		}
	}()
	bus.Pub("topic1", ports.Event{"one", "two"})
	bus.Unsub(ch)
	wg.Wait()
	bus.Shutdown()
}
