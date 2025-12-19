package main

import (
	"log"
	"net/http"
	"os/exec"
	"pagina/internal/adapters/handlers"
	"pagina/internal/adapters/ytdlp"
	"pagina/internal/core/services"
	"strings"
)

func main() {
	// 1. Adapters (Driven)
	ytRepo := ytdlp.NewYtDlpAdapter()

	// 2. Core Service
	dlService := services.NewDownloaderService(ytRepo)

	// 3. Adapter (Driving)
	httpHandler := handlers.NewHTTPHandler(dlService)

	// 4. Router
	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/info", httpHandler.HandleInfo)
	http.HandleFunc("/download", httpHandler.HandleDownload)

	log.Println("Server starting on port 8081...")

	// Verify yt-dlp
	out, err := exec.Command("/usr/bin/yt-dlp", "--version").Output()
	if err != nil {
		log.Printf("WARNING: yt-dlp not found at /usr/bin/yt-dlp: %v", err)
	} else {
		log.Printf("Found yt-dlp version: %s", strings.TrimSpace(string(out)))
	}

	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}
