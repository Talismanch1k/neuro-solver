package llmcore

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const base_url string = "https://openrouter.ai/api/v1"

// const model string = "tngtech/deepseek-r1t-chimera:free"
const model string = "openai/gpt-oss-20b:free"

func LLMQuery(systemPrompt, userPrompt string, temperature float64) string {
	client := openai.NewClient(option.WithBaseURL(base_url))

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
