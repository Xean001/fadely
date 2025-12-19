package youtube

import (
	"io"
	"pagina/internal/core/ports"

	"github.com/kkdai/youtube/v2"
)

type youtubeRepo struct {
	client youtube.Client
}

func NewYouTubeRepository() ports.YouTubeRepository {
	return &youtubeRepo{
		client: youtube.Client{},
	}
}

func (r *youtubeRepo) GetVideo(url string) (*youtube.Video, error) {
	return r.client.GetVideo(url)
}

func (r *youtubeRepo) GetPlaylist(url string) (*youtube.Playlist, error) {
	return r.client.GetPlaylist(url)
}

func (r *youtubeRepo) GetStream(video *youtube.Video, format *youtube.Format) (io.ReadCloser, int64, error) {
	return r.client.GetStream(video, format)
}
