package chatcompletionstream

import (
	"context"
	"errors"
	"github.com/FM-007/chatservice/internal/domain/entities"
	"github.com/FM-007/chatservice/internal/domain/gateway"
	openai "github.com/sashabaranov/go-openai"
	"io"
	"strings"
)

type ChatCompletionConfigInputDTO struct {
	Model                string
	ModelMaxToken        int
	Temperature          float32
	TopP                 float32
	N                    int
	Stop                 []string
	MaxTokens            int
	PresencePenalty      float32
	FrequencyPenalty     float32
	InitialSystemMessage string
}

type ChatCompletionUseCase struct {
	ChatGateway  gateway.ChatGateway
	OpenAiClient *openai.Client
	Stream       chan ChatCompletionOutputDTO
}

type ChatCompletionInputDTO struct {
	ChatID      string
	UserID      string
	UserMessage string
	Config      *ChatCompletionConfigInputDTO
}

type ChatCompletionOutputDTO struct {
	ChatID  string
	UserID  string
	Content string
}

func NewChatCompletionUseCase(chatGateway gateway.ChatGateway, openAiClient *openai.Client, stream chan ChatCompletionOutputDTO) *ChatCompletionUseCase {
	return &ChatCompletionUseCase{
		ChatGateway:  chatGateway,
		OpenAiClient: openAiClient,
	}
}

func (useCase *ChatCompletionUseCase) Execute(ctx context.Context, input ChatCompletionInputDTO) (*ChatCompletionOutputDTO, error) {
	chat, erro := useCase.ChatGateway.FindChatByID(ctx, input.ChatID)
	if erro != nil {
		if erro.Error() == "chat not found" {
			// Create new chat (entity)
			chat, erro = CreateNewChat(input)
			if erro != nil {
				return nil, errors.New("Error creating new chat: " + erro.Error())
			}
			// Save on database
			erro = useCase.ChatGateway.CreateChat(ctx, chat)
			if erro != nil {
				return nil, errors.New("Error persisting new chat: " + erro.Error())
			}
		} else {
			return nil, errors.New("Error fetching existing chat: " + erro.Error())
		}
	}

	userMessage, erro := entities.NewMessage("user", input.UserMessage, chat.Config.Model)
	if erro != nil {
		return nil, errors.New("Error creating user message: " + erro.Error())
	}

	erro = chat.AddMessage(userMessage)
	if erro != nil {
		return nil, errors.New("Error adding new message: " + erro.Error())
	}

	messages := []openai.ChatCompletionMessage{}
	for _, msg := range chat.Messages {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	resp, erro := useCase.OpenAiClient.CreateCompletionStream(
		ctx,
		openai.CompletionRequest{
			Model:            chat.Config.Model.Name,
			Messages:         messages,
			MaxTokens:        chat.Config.MaxTokens,
			Temperature:      chat.Config.Temperature,
			TopP:             chat.Config.TopP,
			PresencePenalty:  chat.Config.PresencePenalty,
			FrequencyPenalty: chat.Config.FrequencyPenalty,
			Stop:             chat.Config.Stop,
			Stream:           true,
		},
	)
	if erro != nil {
		return nil, errors.New("Error creating chat completion: " + erro.Error())
	}

	var fullResponse strings.Builder
	for {
		response, erro := resp.Recv()
		if errors.Is(erro, io.EOF) {
			break
		}
		if erro != nil {
			return nil, errors.New("Error streaming response: " + erro.Error())
		}
		fullResponse.WriteString(response.Choices[0].Delta.Content)
		res := ChatCompletionOutputDTO{
			ChatID:  chat.ID,
			UserID:  input.UserID,
			Content: fullResponse.String(),
		}
		useCase.Stream <- res
	}

	assistant, erro := entities.NewMessage("assistant", fullResponse.String(), chat.Config.Model)
	if erro != nil {
		return nil, errors.New("Error creating assistant message: " + erro.Error())
	}
	erro = chat.AddMessage(assistant)
	if erro != nil {
		return nil, errors.New("Error adding new message: " + erro.Error())
	}
	erro = useCase.ChatGateway.SaveChat(ctx, chat)
	if erro != nil {
		return nil, errors.New("Error saving chat: " + erro.Error())
	}

	return &ChatCompletionOutputDTO{
		ChatID:  chat.ID,
		UserID:  input.UserID,
		Content: fullResponse.String(),
	}, nil
}

func CreateNewChat(input ChatCompletionInputDTO) (*entities.Chat, error) {
	model := entities.NewModel(input.Config.Model, input.Config.ModelMaxToken)
	chatConfig := &entities.ChatConfig{
		Temperature:      input.Config.Temperature,
		TopP:             input.Config.TopP,
		N:                input.Config.N,
		Stop:             input.Config.Stop,
		MaxTokens:        input.Config.MaxTokens,
		PresencePenalty:  input.Config.PresencePenalty,
		FrequencyPenalty: input.Config.FrequencyPenalty,
		Model:            model,
	}

	initialMessage, erro := entities.NewMessage("System", input.Config.InitialSystemMessage, model)
	if erro != nil {
		return nil, errors.New("Error creating initial message: " + erro.Error())
	}

	chat, erro := entities.NewChat(input.UserID, initialMessage, chatConfig)
	if erro != nil {
		return nil, errors.New("Error creating new chat: " + erro.Error())
	}
	return chat, nil
}
