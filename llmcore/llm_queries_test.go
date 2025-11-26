package llmcore

import (
	"os"
	"testing"
)

func TestParseStringList_ValidJSON(t *testing.T) {
	input := `["apple", "banana", "cherry"]`
	expected := []string{"apple", "banana", "cherry"}

	result, err := ParseStringList(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != len(expected) {
		t.Fatalf("length mismatch: got %d, want %d", len(result), len(expected))
	}

	for i, v := range expected {
		if result[i] != v {
			t.Errorf("element %d: got %q, want %q", i, result[i], v)
		}
	}
}

func TestParseStringList_EmptyArray(t *testing.T) {
	input := `[]`

	result, err := ParseStringList(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Fatalf("expected empty slice, got %d elements", len(result))
	}
}

func TestParseStringList_InvalidJSON(t *testing.T) {
	input := `not a json`

	_, err := ParseStringList(input)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestParseStringList_WrongType(t *testing.T) {
	input := `{"key": "value"}`

	_, err := ParseStringList(input)
	if err == nil {
		t.Fatal("expected error for wrong JSON type, got nil")
	}
}

func TestParseStringList_Unicode(t *testing.T) {
	input := `["привет", "мир", "🚀"]`
	expected := []string{"привет", "мир", "🚀"}

	result, err := ParseStringList(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, v := range expected {
		if result[i] != v {
			t.Errorf("element %d: got %q, want %q", i, result[i], v)
		}
	}
}

// TestLLMQuery_Connection проверяет, что API доступен и возвращает ответ.
// Этот тест пропускается, если не установлена переменная окружения OPENROUTER_API_KEY
// или если передан флаг -short.
func TestLLMQuery_Connection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Проверяем наличие API ключа
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("skipping: OPENAI_API_KEY not set")
	}

	// Простой запрос для проверки соединения
	systemPrompt := "You are a helpful assistant. Respond with exactly one word."
	userPrompt := "Say 'pong'"

	result := LLMQuery(systemPrompt, userPrompt, 0.1)

	if result == "" {
		t.Fatal("expected non-empty response from LLM API")
	}

	t.Logf("LLM response: %s", result)
}
