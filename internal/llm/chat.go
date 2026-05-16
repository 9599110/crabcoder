package llm

import (
	"context"

	"github.com/crabcoder/crabcoder/pkg/model"
)

type ChatSession struct {
	provider LLMProvider
	history  []model.Message
}

func NewChatSession(provider LLMProvider) *ChatSession {
	return &ChatSession{
		provider: provider,
		history:  make([]model.Message, 0),
	}
}

func (s *ChatSession) Send(ctx context.Context, content string) (string, error) {
	s.history = append(s.history, model.Message{
		Role:    model.RoleUser,
		Content: content,
	})

	resp, err := s.provider.Chat(ctx, s.history, &ChatOptions{})
	if err != nil {
		return "", err
	}

	s.history = append(s.history, model.Message{
		Role:    model.RoleAssistant,
		Content: resp.Content,
	})

	return resp.Content, nil
}

func (s *ChatSession) History() []model.Message {
	return s.history
}

func (s *ChatSession) Clear() {
	s.history = make([]model.Message, 0)
}
