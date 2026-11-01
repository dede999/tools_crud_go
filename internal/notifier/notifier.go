package notifier

import (
	"context"
	"sync"
	"time"

	"github.com/dede999/tools_crud_go/internal/domain"
)

type DeviceStateChangeEvent struct {
	DeviceID      string             `json:"device_id"`
	PreviousState domain.DeviceState `json:"previous_state"`
	NewState      domain.DeviceState `json:"new_state"`
	Timestamp     time.Time          `json:"timestamp"`
}

type Notifier interface {
	Subscribe(ctx context.Context) (<-chan DeviceStateChangeEvent, func())
	Publish(event DeviceStateChangeEvent)
}

type EventBroker struct {
	mu      sync.RWMutex
	clients map[chan DeviceStateChangeEvent]struct{}
}

func NewBroker() *EventBroker {
	return &EventBroker{
		clients: make(map[chan DeviceStateChangeEvent]struct{}),
	}
}

func (b *EventBroker) Subscribe(ctx context.Context) (<-chan DeviceStateChangeEvent, func()) {
	ch := make(chan DeviceStateChangeEvent, 16)

	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.clients, ch)
			close(ch)
			b.mu.Unlock()
		})
	}

	go func() {
		<-ctx.Done()
		unsubscribe()
	}()

	return ch, unsubscribe
}

func (b *EventBroker) Publish(event DeviceStateChangeEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.clients {
		select {
		case ch <- event:
		default:
			// Non-blocking write: if a subscriber's buffer is full, don't stall the publisher
		}
	}
}

var _ Notifier = (*EventBroker)(nil)
