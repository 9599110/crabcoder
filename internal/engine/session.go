package engine

import (
	"sync"
	"time"

	"github.com/crabcoder/crabcoder/internal/event"
)

type SessionState string

const (
	SessionIdle       SessionState = "idle"
	SessionParsing    SessionState = "parsing"
	SessionScheduling SessionState = "scheduling"
	SessionExecuting  SessionState = "executing"
	SessionWaiting    SessionState = "waiting"
	SessionCompleted  SessionState = "completed"
	SessionError      SessionState = "error"
)

type Session struct {
	mu        sync.RWMutex
	state     SessionState
	requestID string
	events    *event.Bus
	startedAt time.Time
}

func NewSession(bus *event.Bus) *Session {
	return &Session{
		state:  SessionIdle,
		events: bus,
	}
}

func (s *Session) State() SessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *Session) Transition(newState SessionState) {
	s.mu.Lock()
	old := s.state
	s.state = newState
	s.mu.Unlock()

	if s.events != nil {
		s.events.Publish(event.Event{
			Type: event.SessionState,
			Data: map[string]any{
				"from": string(old),
				"to":   string(newState),
			},
		})
	}
}

func (s *Session) Start(requestID string) {
	s.mu.Lock()
	s.requestID = requestID
	s.startedAt = time.Now()
	s.mu.Unlock()
	s.Transition(SessionParsing)
}
