package entities

import (
	"errors"
	"github.com/google/uuid"
	tiktoken_go "github.com/j178/tiktoken-go"
	"time"
)

type Message struct {
	ID        string
	Role      string
	Content   string
	Tokens    int
	Model     *Model
	CreatedAt time.Time
}

func NewMessage(role, content string, model *Model) (*Message, error) {
	totalTokens := tiktoken_go.CountTokens(model.GetModelName(), content)
	message := &Message{
		ID:        uuid.New().String(),
		Role:      role,
		Content:   content,
		Tokens:    totalTokens,
		Model:     model,
		CreatedAt: time.Time{},
	}
	if erro := message.Validate(); erro != nil {
		return nil, erro
	}
	return message, nil
}

func (message *Message) Validate() error {
	if message.Role != "user" && message.Role != "system" && message.Role != "assistant" {
		return errors.New("Invalid Role")
	}
	if message.Content == "" {
		return errors.New("Content is empty!")
	}
	if message.CreatedAt.IsZero() {
		return errors.New("Invalid created_at")
	}
	return nil
}

func (message *Message) GetQtdToken() int {
	return message.Tokens
}
