package ports

import (
	"io"
	"pagina/internal/core/domain"

	"github.com/kkdai/youtube/v2"
)

type DownloaderService interface {
	GetVideoInfo(url string) (*domain.VideoInfo, error)
	GetPlaylistInfo(url string) (*domain.PlaylistInfo, error)
	DownloadVideo(url, format, quality string) (io.ReadCloser, string, string, error) // Returns stream, filename, content-type, error
}

type YouTubeRepository interface {
	GetVideo(url string) (*youtube.Video, error)
	GetPlaylist(url string) (*youtube.Playlist, error)
	GetStream(video *youtube.Video, format *youtube.Format) (io.ReadCloser, int64, error)
}

type MediaProcessor interface {
	MuxVideoAudio(videoFile, audioFile string) (string, error)
	ConvertToMP3(audioFile string) (string, error)
}
