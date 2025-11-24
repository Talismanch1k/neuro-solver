package main

import (
	"embed"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"

	webview "github.com/webview/webview_go"
)

// 1. КОМПИЛЯЦИЯ ВМЕСТЕ:
// Эта строчка говорит Go: "Засунь эти файлы внутрь EXE"
//
//go:embed index.html style.css
var assets embed.FS

func main() {
	// 2. СОЗДАЕМ МОСТИК (Сервер):
	// Создаем локальный сервер, который раздает файлы из памяти (assets)
	// Порт ":0" означает "дай любой свободный порт", чтобы не было конфликтов
	if runtime.GOOS == "linux" {
		os.Setenv("WEBKIT_DISABLE_COMPOSITING_MODE", "1")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	// Запускаем сервер в фоновом режиме
	go http.Serve(ln, http.FileServer(http.FS(assets)))

	// 3. ЗАПУСКАЕМ ОКНО:
	w := webview.New(true)
	defer w.Destroy()
	w.SetTitle("Neural Solver")
	w.SetSize(500, 700, webview.HintNone)

	// API функция (Backend логика)
	w.Bind("solveProblem", func(text string) string {
		return "Go Backend processed: " + text
	})

	// Открываем наш внутренний сайт
	// http://127.0.0.1:xxxxx/index.html
	w.Navigate("http://" + ln.Addr().String() + "/index.html")

	w.Run()
}
