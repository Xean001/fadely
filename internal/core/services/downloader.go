package services

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"pagina/internal/core/domain"
	"pagina/internal/core/ports"
)

type downloaderService struct {
	ytRepo ports.YouTubeRepository
}

func NewDownloaderService(yt ports.YouTubeRepository) ports.DownloaderService {
	return &downloaderService{
		ytRepo: yt,
	}
}

func (s *downloaderService) GetVideoInfo(url string) (*domain.VideoInfo, error) {
	return s.ytRepo.GetVideo(url)
}

func (s *downloaderService) GetPlaylistInfo(urlInput string) (*domain.PlaylistInfo, error) {
	// Try to extract list ID if needed, similar to before, or trust yt-dlp to handle common URLs.
	// yt-dlp is smart, but helping it with the correct URL doesn't hurt.
	targetURL := urlInput
	u, err := url.Parse(urlInput)
	if err == nil {
		q := u.Query()
		if listID := q.Get("list"); listID != "" {
			targetURL = "https://www.youtube.com/playlist?list=" + listID
		}
	}

	return s.ytRepo.GetPlaylist(targetURL)
}

func (s *downloaderService) DownloadVideo(url, format, quality string) (io.ReadCloser, string, string, error) {
	// Fetch video meta to get title for filename if needed?
	// The adapter returns the file path. We can get the filename from there or metadata.

	path, err := s.ytRepo.Download(url, format, quality)
	if err != nil {
		return nil, "", "", err
	}

	// Create a reader that deletes the file on close
	reader, _, _, err := newDeleteOnCloseReader(path)
	if err != nil {
		return nil, "", "", err
	}

	// Determine content type and filename
	// We can guess from extension
	filename := "video.mp4"
	contentType := "video/mp4"
	if format == "mp3" {
		filename = "audio.mp3"
		contentType = "audio/mpeg"
	}

	// Better: Get metadata to have real title
	// This adds a second call but ensures nice filenames
	info, err := s.ytRepo.GetVideo(url)
	if err == nil {
		filename = fmt.Sprintf("%s.%s", info.Title, format)
		if format == "" {
			format = "mp4"
		} // default
	}

	return reader, filename, contentType, nil
}

// Custom ReadCloser that deletes the file when closed
type deleteOnCloseReader struct {
	*os.File
}

func newDeleteOnCloseReader(path string) (io.ReadCloser, string, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", "", err
	}
	return &deleteOnCloseReader{File: f}, "", "application/octet-stream", nil
}

func (d *deleteOnCloseReader) Close() error {
	path := d.File.Name()
	err := d.File.Close()
	_ = os.Remove(path) // Delete file on close
	return err
}
