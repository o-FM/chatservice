package repository

import (
	"context"
	"database/sql"
	"errors"
	"github.com/FM-007/chatservice/internal/domain/entities"
	"github.com/FM-007/chatservice/internal/infra/db"
	"time"
)

type ChatRepositoryMySQL struct {
	DB      *sql.DB
	Queries *db.Queries
}

func NewChatRepositoryMySQL(dbt *sql.DB) *ChatRepositoryMySQL {
	return &ChatRepositoryMySQL{
		DB:      dbt,
		Queries: db.New(dbt),
	}
}

func (chatRepository *ChatRepositoryMySQL) CreateChat(ctx context.Context, chat *entities.Chat) error {
	err := chatRepository.Queries.CreateChat(
		ctx, db.CreateChatParams{
			ID:               chat.ID,
			UserID:           chat.UserID,
			InitialMessageID: chat.InitialSystemMessage.Content,
			Status:           chat.Status,
			TokenUsage:       int32(chat.TokenUsage),
			Model:            chat.Config.Model.Name,
			ModelMaxTokens:   int32(chat.Config.Model.MaxTokens),
			Temperature:      float64(chat.Config.Temperature),
			TopP:             float64(chat.Config.TopP),
			N:                int32(chat.Config.N),
			Stop:             chat.Config.Stop[0],
			MaxTokens:        int32(chat.Config.MaxTokens),
			PresencePenalty:  float64(chat.Config.PresencePenalty),
			FrequencyPenalty: float64(chat.Config.FrequencyPenalty),
			CreatedAt:        time.Time{},
			UpdatedAt:        time.Time{},
		},
	)
	if err != nil {
		return err
	}

	err = chatRepository.Queries.AddMessage(
		ctx, db.AddMessageParams{
			ID:        chat.InitialSystemMessage.ID,
			ChatID:    chat.ID,
			Content:   chat.InitialSystemMessage.Content,
			Role:      chat.InitialSystemMessage.Role,
			Tokens:    int32(chat.InitialSystemMessage.Tokens),
			CreatedAt: chat.InitialSystemMessage.CreatedAt,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (chatRepository *ChatRepositoryMySQL) FindChatByID(ctx context.Context, chatID string) (*entities.Chat, error) {
	chat := &entities.Chat{}
	resp, erro := chatRepository.Queries.FindChatByID(ctx, chatID)
	if erro != nil {
		return nil, errors.New("Chat not found!")
	}

	chat.ID = resp.ID
	chat.UserID = resp.UserID
	chat.Status = resp.Status
	chat.TokenUsage = int(resp.TokenUsage)
	chat.Config = &entities.ChatConfig{
		Model: &entities.Model{
			Name:      resp.Model,
			MaxTokens: int(resp.ModelMaxTokens),
		},
		Temperature:      float32(resp.Temperature),
		TopP:             float32(resp.TopP),
		N:                int(resp.N),
		Stop:             []string{resp.Stop},
		MaxTokens:        int(resp.MaxTokens),
		PresencePenalty:  float32(resp.PresencePenalty),
		FrequencyPenalty: float32(resp.FrequencyPenalty),
	}

	messages, erro := chatRepository.Queries.FindMessagesByChatID(ctx, chatID)
	if erro != nil {
		return nil, erro
	}
	for _, message := range messages {
		chat.Messages = append(chat.Messages, &entities.Message{
			ID:        message.ID,
			Content:   message.Content,
			Role:      message.Role,
			Tokens:    int(message.Tokens),
			Model:     &entities.Model{Name: message.Model},
			CreatedAt: message.CreatedAt,
		})
	}

	erasedMessages, erro := chatRepository.Queries.FindErasedMessagesByChatID(ctx, chatID)
	if erro != nil {
		return nil, erro
	}
	for _, erasedMessage := range erasedMessages {
		chat.Messages = append(chat.Messages, &entities.Message{
			ID:        erasedMessage.ID,
			Content:   erasedMessage.Content,
			Role:      erasedMessage.Role,
			Tokens:    int(erasedMessage.Tokens),
			Model:     &entities.Model{Name: erasedMessage.Model},
			CreatedAt: erasedMessage.CreatedAt,
		})
	}

	return chat, nil
}

func (r *ChatRepositoryMySQL) SaveChat(ctx context.Context, chat *entities.Chat) error {
	params := db.SaveChatParams{
		ID:               chat.ID,
		UserID:           chat.UserID,
		Status:           chat.Status,
		TokenUsage:       int32(chat.TokenUsage),
		Model:            chat.Config.Model.Name,
		ModelMaxTokens:   int32(chat.Config.Model.MaxTokens),
		Temperature:      float64(chat.Config.Temperature),
		TopP:             float64(chat.Config.TopP),
		N:                int32(chat.Config.N),
		Stop:             chat.Config.Stop[0],
		MaxTokens:        int32(chat.Config.MaxTokens),
		PresencePenalty:  float64(chat.Config.PresencePenalty),
		FrequencyPenalty: float64(chat.Config.FrequencyPenalty),
		UpdatedAt:        time.Now(),
	}

	err := r.Queries.SaveChat(
		ctx,
		params,
	)
	if err != nil {
		return err
	}
	// delete messages
	err = r.Queries.DeleteChatMessages(ctx, chat.ID)
	if err != nil {
		return err
	}
	// delete erased messages
	err = r.Queries.DeleteErasedChatMessages(ctx, chat.ID)
	if err != nil {
		return err
	}
	// save messages
	i := 0
	for _, message := range chat.Messages {
		err = r.Queries.AddMessage(
			ctx,
			db.AddMessageParams{
				ID:        message.ID,
				ChatID:    chat.ID,
				Content:   message.Content,
				Role:      message.Role,
				Tokens:    int32(message.Tokens),
				Model:     chat.Config.Model.Name,
				CreatedAt: message.CreatedAt,
				OrderMsg:  int32(i),
				Erased:    false,
			},
		)
		if err != nil {
			return err
		}
		i++
	}
	// save erased messages
	i = 0
	for _, message := range chat.EraseMessages {
		err = r.Queries.AddMessage(
			ctx,
			db.AddMessageParams{
				ID:        message.ID,
				ChatID:    chat.ID,
				Content:   message.Content,
				Role:      message.Role,
				Tokens:    int32(message.Tokens),
				Model:     chat.Config.Model.Name,
				CreatedAt: message.CreatedAt,
				OrderMsg:  int32(i),
				Erased:    true,
			},
		)
		if err != nil {
			return err
		}
		i++
	}
	return nil
}
