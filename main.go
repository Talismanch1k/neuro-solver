package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"neurosolver/llmcore"
	"neurosolver/resolution"
	"os"
	"runtime"

	webview "github.com/webview/webview_go"
)

//go:embed assets/*
var assets embed.FS

func main() {
	// Disable WebKit compositing mode on Linux to avoid rendering issues
	if runtime.GOOS == "linux" {
		os.Setenv("WEBKIT_DISABLE_COMPOSITING_MODE", "1")
		os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
		os.Setenv("GDK_BACKEND", "x11")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:51115")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	// launch server in background
	go http.Serve(ln, http.FileServer(http.FS(assets)))

	// launch window
	w := webview.New(true)
	defer w.Destroy()
	w.SetTitle("Neural Solver")
	w.SetSize(500, 700, webview.HintNone)

	// API функция (Backend логика)
	w.Bind("solveProblemAsync", func(text string, showLog bool, callbackId string) {
		// Запускаем в отдельной горутине
		go func() {
			result := llmcore.LLMQuery(llmcore.ParsingPrompt, text, 0.2)
			parsedResult, _ := llmcore.ParseStringList(result)

			engine := resolution.NewResolutionEngine()
			engine.ParseInput(parsedResult)
			shortLog := engine.Prove().ShortLog

			explanation := llmcore.LLMQuery(llmcore.ExplanationPrompt, shortLog, 0.4)

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
	})

	w.Navigate("http://" + ln.Addr().String() + "/assets/index.html")

	w.Run()
}
