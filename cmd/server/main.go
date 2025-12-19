package main

import (
	"log"
	"net/http"
	"pagina/internal/adapters/ffmpeg"
	"pagina/internal/adapters/handlers"
	"pagina/internal/adapters/youtube"
	"pagina/internal/core/services"
)

func main() {
	// 1. Adapters (Driven)
	ytRepo := youtube.NewYouTubeRepository()
	mediaProc := ffmpeg.NewFFmpegProcessor()

	// 2. Core Service
	dlService := services.NewDownloaderService(ytRepo, mediaProc)

	// 3. Adapter (Driving)
	httpHandler := handlers.NewHTTPHandler(dlService)

	// 4. Router
	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/info", httpHandler.HandleInfo)
	http.HandleFunc("/download", httpHandler.HandleDownload)

	log.Println("Server starting on port 8081...")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}
