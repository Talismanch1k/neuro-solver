package llmcore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const base_url string = "https://openrouter.ai/api/v1"

// const model string = "tngtech/deepseek-r1t-chimera:free"
const model string = "openai/gpt-oss-20b:free"

var embeddedAPIKey string = ""

func getAPIKey() string {
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		return key
	}
	return embeddedAPIKey
}

func LLMQuery(systemPrompt, userPrompt string, temperature float64) string {
	apiKey := getAPIKey()
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
		panic(err.Error())
	}

	return resp.Choices[0].Message.Content
}

func ParseStringList(input string) ([]string, error) {
	var result []string

	err := json.Unmarshal([]byte(input), &result)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	return result, nil
}
