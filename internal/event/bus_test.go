package event

import "testing"

func TestPublishSubscribe(t *testing.T) {
	bus := NewBus()

	ch := bus.Subscribe(TaskCompleted)

	bus.Publish(Event{Type: TaskCompleted, Data: map[string]any{"task_id": "1"}})

	select {
	case evt := <-ch:
		if evt.Type != TaskCompleted {
			t.Fatalf("expected TaskCompleted, got %s", evt.Type)
		}
		if evt.Data["task_id"] != "1" {
			t.Fatalf("expected task_id=1, got %v", evt.Data["task_id"])
		}
	default:
		t.Fatal("expected event, got none")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := NewBus()

	ch1 := bus.Subscribe(TaskStarted)
	ch2 := bus.Subscribe(TaskStarted)

	bus.Publish(Event{Type: TaskStarted, Data: map[string]any{"task_id": "42"}})

	for _, ch := range []<-chan Event{ch1, ch2} {
		select {
		case evt := <-ch:
			if evt.Data["task_id"] != "42" {
				t.Fatalf("expected task_id=42")
			}
		default:
			t.Fatal("expected event")
		}
	}
}

func TestDoesNotCrossTypes(t *testing.T) {
	bus := NewBus()

	ch := bus.Subscribe(TaskCompleted)
	bus.Publish(Event{Type: TaskStarted, Data: map[string]any{"task_id": "99"}})

	select {
	case <-ch:
		t.Fatal("TaskCompleted subscriber should not receive TaskStarted events")
	default:
		// expected
	}
}
