package backend

import (
	"encoding/json"
	"fmt"
	"neurosolver/llmcore"
	"neurosolver/resolution"

	webview "github.com/webview/webview_go"
)

var (
	cacheText        string
	cacheShortLog    string
	cacheExplanation string
)

// SolveProblemHandler возвращает функцию-обработчик для решения логических задач
func SolveProblemHandler(w webview.WebView) func(text string, showLog bool, callbackId string) {
	return func(text string, showLog bool, callbackId string) {
		// Запускаем в отдельной горутине
		go func() {
			// Вспомогательная функция для отправки ошибки в UI
			sendError := func(errMsg string) {
				w.Dispatch(func() {
					escaped, _ := json.Marshal("❌ Ошибка: " + errMsg)
					w.Eval(fmt.Sprintf("window._resolveCallback('%s', %s)", callbackId, escaped))
				})
			}

			// Проверяем кэш - если текст тот же, просто переформатируем результат
			if cacheText == text && cacheShortLog != "" && cacheExplanation != "" {
				fmt.Println("CACHED VALUE!!!")
				var finalResult string
				if showLog {
					finalResult = "=== Лог движка резолюций ===\n" + cacheShortLog + "\n\n=== Объяснение ===\n" + cacheExplanation
				} else {
					finalResult = cacheExplanation
				}

				w.Dispatch(func() {
					escaped, _ := json.Marshal(finalResult)
					w.Eval(fmt.Sprintf("window._resolveCallback('%s', %s)", callbackId, escaped))
				})
				return
			}

			// Шаг 1: Парсинг текста через LLM
			result, err := llmcore.LLMQuery(llmcore.ParsingPrompt, text, 0.2)
			fmt.Println("LLM Parsed:", result)
			if err != nil {
				sendError(err.Error())
				return
			}

			parsedResult, err := llmcore.ParseStringList(result)
			fmt.Println("After parse json:", parsedResult)
			if err != nil {
				sendError("Не удалось распознать логические формулы: " + err.Error())
				return
			}

			if len(parsedResult) == 0 {
				sendError("LLM вернул пустой результат. Попробуйте переформулировать задачу.")
				return
			}

			// Шаг 2: Запуск движка резолюций
			engine := resolution.NewResolutionEngine()
			engine.ParseInput(parsedResult)
			proofResult := engine.Prove()
			shortLog := proofResult.ShortLog
			fmt.Println("SHORT LOG:", shortLog)

			// Шаг 3: Генерация объяснения через LLM
			explanation, err := llmcore.LLMQuery(llmcore.ExplanationPrompt, shortLog, 1)
			fmt.Println("EXPLANATION:", explanation)
			if err != nil {
				// Если не удалось получить объяснение, показываем хотя бы лог
				explanation = "(Не удалось сгенерировать объяснение: " + err.Error() + ")"
			}

			// Сохраняем в кэш
			cacheText = text
			cacheShortLog = shortLog
			cacheExplanation = explanation

			// Формируем результат в зависимости от флага
			var finalResult string
			if showLog {
				finalResult = "=== Лог движка резолюций ===\n" + shortLog + "\n\n=== Объяснение ===\n" + explanation
			} else {
				finalResult = explanation
			}

			// Возвращаем результат через JS callback
			w.Dispatch(func() {
				// Экранируем кавычки и переносы строк в результате
				escaped, _ := json.Marshal(finalResult)
				w.Eval(fmt.Sprintf("window._resolveCallback('%s', %s)", callbackId, escaped))
			})
		}()
	}
}
