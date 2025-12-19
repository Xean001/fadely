package ports

import (
	"io"
	"pagina/internal/core/domain"
)

type DownloaderService interface {
	GetVideoInfo(url string) (*domain.VideoInfo, error)
	GetPlaylistInfo(url string) (*domain.PlaylistInfo, error)
	DownloadVideo(url, format, quality string) (io.ReadCloser, string, string, error) // Returns stream, filename, content-type, error
}

type YouTubeRepository interface {
	GetVideo(url string) (*domain.VideoInfo, error)
	GetPlaylist(url string) (*domain.PlaylistInfo, error)
	// Download downloads the video to a local path and returns the path
	Download(url, format, quality string) (string, error)
}
