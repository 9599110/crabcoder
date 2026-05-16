package event

import "sync"

type EventType string

const (
	TaskStarted      EventType = "task.started"
	TaskCompleted    EventType = "task.completed"
	TaskFailed       EventType = "task.failed"
	ProgressUpdate   EventType = "progress.update"
	ApprovalRequired EventType = "approval.required"
	SessionState     EventType = "session.state"
)

type Event struct {
	Type EventType
	Data map[string]any
}

type Bus struct {
	mu          sync.RWMutex
	subscribers map[EventType][]chan Event
}

func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[EventType][]chan Event),
	}
}

func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.subscribers[event.Type] {
		select {
		case ch <- event:
		default:
			// drop if subscriber is not consuming
		}
	}
}

func (b *Bus) Subscribe(eventType EventType) <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event, 64)
	b.subscribers[eventType] = append(b.subscribers[eventType], ch)
	return ch
}
