package context

import "github.com/crabcoder/crabcoder/pkg/model"

type MessageHistory struct {
	messages []model.Message
}

func NewMessageHistory() *MessageHistory {
	return &MessageHistory{
		messages: make([]model.Message, 0),
	}
}

func (h *MessageHistory) Add(msg model.Message) {
	h.messages = append(h.messages, msg)
}

func (h *MessageHistory) All() []model.Message {
	return h.messages
}

func (h *MessageHistory) Len() int {
	return len(h.messages)
}
