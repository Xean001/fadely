package domain

type PlaylistInfo struct {
	ID     string      `json:"id"`
	Title  string      `json:"title"`
	Author string      `json:"author"`
	Videos []VideoInfo `json:"videos"`
}

type VideoInfo struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Author      string        `json:"author"`
	Description string        `json:"description"`
	Duration    string        `json:"duration"`
	Thumbnail   string        `json:"thumbnail"`
	Formats     []VideoFormat `json:"formats"`
}

type VideoFormat struct {
	Label string `json:"label"` // e.g. "1080p"
	Itag  int    `json:"itag"`
	Type  string `json:"type"` // "video/mp4; codecs=..."
}

type DownloadRequest struct {
	URL     string `json:"url"`
	Format  string `json:"format"`  // "mp4" or "mp3"
	Quality string `json:"quality"` // itag for mp4
}
