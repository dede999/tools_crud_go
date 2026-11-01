package notifier_test

import (
	"context"
	"testing"
	"time"

	"github.com/dede999/tools_crud_go/internal/domain"
	"github.com/dede999/tools_crud_go/internal/notifier"
)

func TestEventBroker_PublishAndSubscribe(t *testing.T) {
	broker := notifier.NewBroker()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eventsCh, unsubscribe := broker.Subscribe(ctx)
	defer unsubscribe()

	event := notifier.DeviceStateChangeEvent{
		DeviceID:      "dev-123",
		PreviousState: domain.StateAvailable,
		NewState:      domain.StateInUse,
		Timestamp:     time.Now().UTC(),
	}

	broker.Publish(event)

	select {
	case received := <-eventsCh:
		if received.DeviceID != event.DeviceID {
			t.Errorf("expected device ID %q, got %q", event.DeviceID, received.DeviceID)
		}
		if received.PreviousState != event.PreviousState {
			t.Errorf("expected previous state %q, got %q", event.PreviousState, received.PreviousState)
		}
		if received.NewState != event.NewState {
			t.Errorf("expected new state %q, got %q", event.NewState, received.NewState)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for event on subscription channel")
	}
}

func TestEventBroker_Unsubscribe(t *testing.T) {
	broker := notifier.NewBroker()

	ctx, cancel := context.WithCancel(context.Background())
	eventsCh, unsubscribe := broker.Subscribe(ctx)

	// Unsubscribe explicitly
	unsubscribe()

	event := notifier.DeviceStateChangeEvent{
		DeviceID:      "dev-456",
		PreviousState: domain.StateInUse,
		NewState:      domain.StateInactive,
		Timestamp:     time.Now().UTC(),
	}

	broker.Publish(event)

	select {
	case _, ok := <-eventsCh:
		if ok {
			t.Errorf("expected channel to be closed or empty after unsubscribe")
		}
	default:
		// channel empty as expected
	}
	cancel()
}
