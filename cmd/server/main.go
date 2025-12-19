package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"pagina/internal/adapters/handlers"
	"pagina/internal/adapters/ytdlp"
	"pagina/internal/core/services"
	"strings"
)

const cookiesPath = "/app/data/cookies.txt"

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

	// Verify Cookies
	if _, err := os.Stat(cookiesPath); err == nil {
		content, _ := os.ReadFile(cookiesPath)
		firstLine := ""
		if len(content) > 0 {
			firstLine = strings.Split(string(content), "\n")[0]
		}
		log.Printf("SUCCESS: Cookie file detected. Size: %d bytes. First line: %q", len(content), firstLine)
		if !strings.Contains(firstLine, "Netscape") {
			log.Printf("WARNING: Cookie file might be invalid! It should start with '# Netscape HTTP Cookie File'")
		}
	} else {
		log.Printf("INFO: No cookie file found at %s. Using automated bypass methods.", cookiesPath)
	}

	// Verify yt-dlp
	out, err := exec.Command("/usr/bin/yt-dlp", "--version").Output()
	if err != nil {
		log.Printf("WARNING: yt-dlp not found at /usr/bin/yt-dlp: %v", err)
	} else {
		log.Printf("Found yt-dlp version: %s", strings.TrimSpace(string(out)))
	}

	// Verify Node.js (required for YouTube challenges)
	nodeOut, nodeErr := exec.Command("node", "--version").Output()
	if nodeErr != nil {
		log.Printf("WARNING: Node.js not found in PATH: %v. YouTube challenges might fail.", nodeErr)
	} else {
		log.Printf("Found Node.js version: %s", strings.TrimSpace(string(nodeOut)))
	}

	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}
