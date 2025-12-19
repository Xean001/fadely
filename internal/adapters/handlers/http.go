package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"pagina/internal/core/domain"
	"pagina/internal/core/ports"
)

type HTTPHandler struct {
	service ports.DownloaderService
}

func NewHTTPHandler(s ports.DownloaderService) *HTTPHandler {
	return &HTTPHandler{service: s}
}

func (h *HTTPHandler) HandleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req domain.DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Heuristic: Check for "list="
	// Note: kkdai/youtube validates URLs rigorously.
	// We try playlist first if it looks like one, or try both.

	// Response Wrapper
	type InfoResponse struct {
		Type     string               `json:"type"` // "video" or "playlist"
		Video    *domain.VideoInfo    `json:"video,omitempty"`
		Playlist *domain.PlaylistInfo `json:"playlist,omitempty"`
	}

	// Try Playlist first if url contains list
	// This is a simple heuristic, but effective for typical YouTube URLs.
	// If it fails, fallback to video.

	var resp InfoResponse

	// Try getting playlist info
	playlist, errPl := h.service.GetPlaylistInfo(req.URL)
	if errPl == nil && playlist != nil {
		resp.Type = "playlist"
		resp.Playlist = playlist
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Fallback to Video
	info, err := h.service.GetVideoInfo(req.URL)
	if err != nil {
		log.Printf("Error getting info: %v (Playlist error: %v)", err, errPl)
		http.Error(w, fmt.Sprintf("Error getting info: %v", err), http.StatusInternalServerError)
		return
	}

	resp.Type = "video"
	resp.Video = info

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *HTTPHandler) HandleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req domain.DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	stream, filename, contentType, err := h.service.DownloadVideo(req.URL, req.Format, req.Quality)
	if err != nil {
		log.Printf("Error downloading: %v", err)
		http.Error(w, fmt.Sprintf("Error downloading video: %v", err), http.StatusInternalServerError)
		return
	}
	defer stream.Close()

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	if _, err := io.Copy(w, stream); err != nil {
		log.Printf("Error streaming response: %v", err)
	}
}
