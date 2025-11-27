package main

import (
	"embed"
	"log"
	"net"
	"net/http"
	"neurosolver/backend"
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

	// Запуск статического файлового сервера
	go http.Serve(ln, http.FileServer(http.FS(assets)))

	// Запуск окна
	w := webview.New(true)
	defer w.Destroy()
	w.SetTitle("Neural Solver")
	w.SetSize(500, 700, webview.HintNone)

	// API функция (Backend логика)
	w.Bind("solveProblemAsync", backend.SolveProblemHandler(w))

	w.Navigate("http://" + ln.Addr().String() + "/assets/index.html")

	w.Run()
}
