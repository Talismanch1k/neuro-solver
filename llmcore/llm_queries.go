package llmcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const base_url string = "https://api.cerebras.ai/v1"
const model string = "gpt-oss-120b"

var embeddedAPIKey string = ""

// Ошибки LLM
var (
	ErrRateLimitExceeded = errors.New("превышен лимит запросов API")
	ErrAPIKeyMissing     = errors.New("API ключ не найден (установите переменную OPENAI_API_KEY)")
	ErrEmptyResponse     = errors.New("получен пустой ответ от LLM")
)

func getAPIKey() string {
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		return key
	}
	return embeddedAPIKey
}

// LLMQuery выполняет запрос к LLM и возвращает результат или ошибку
func LLMQuery(systemPrompt, userPrompt string, temperature float64) (string, error) {
	apiKey := getAPIKey()
	if apiKey == "" {
		return "", ErrAPIKeyMissing
	}

	client := openai.NewClient(option.WithBaseURL(base_url), option.WithAPIKey(apiKey))

	resp, err := client.Chat.Completions.New(context.TODO(),
		openai.ChatCompletionNewParams{
			Model:       model,
			Temperature: openai.Float(temperature),
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(systemPrompt),
				openai.UserMessage(userPrompt),
			},
		})

	if err != nil {
		// Проверяем на rate limit (429)
		errStr := err.Error()
		if strings.Contains(errStr, "429") || strings.Contains(errStr, "Rate limit") {
			return "", ErrRateLimitExceeded
		}
		return "", fmt.Errorf("ошибка API: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return "", ErrEmptyResponse
	}

	return resp.Choices[0].Message.Content, nil
}

func ParseStringList(input string) ([]string, error) {
	var result []string

	err := json.Unmarshal([]byte(input), &result)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	return result, nil
}
