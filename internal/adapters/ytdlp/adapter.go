package ytdlp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"pagina/internal/core/domain"
	"pagina/internal/core/ports"
	"strings"
)

type ytDlpAdapter struct{}

func NewYtDlpAdapter() ports.YouTubeRepository {
	return &ytDlpAdapter{}
}

// Internal struct to match yt-dlp JSON output
type ytDlpJSON struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Uploader    string  `json:"uploader"`
	Description string  `json:"description"`
	Duration    float64 `json:"duration"`
	Thumbnail   string  `json:"thumbnail"`
	WebpageURL  string  `json:"webpage_url"`
	Formats     []struct {
		FormatID   string `json:"format_id"`
		Ext        string `json:"ext"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		VCodec     string `json:"vcodec"`
		ACodec     string `json:"acodec"`
		FormatNote string `json:"format_note"`
	} `json:"formats"`
	Entries []ytDlpJSON `json:"entries"` // For playlists
}

const cookiesPath = "/app/data/cookies.txt"

func (a *ytDlpAdapter) getBaseArgs() []string {
	args := []string{
		"--no-check-certificates",
		"--no-warnings",
		"--force-ipv4",
		"--ignore-config",
		// Using ONLY 'ios' is currently the most powerful bypass for bot detection on VPS
		"--extractor-args", "youtube:player-client=ios",
	}

	// Use cookies if the file exists
	if _, err := os.Stat(cookiesPath); err == nil {
		args = append(args, "--cookies", cookiesPath)
	}

	return args
}

func (a *ytDlpAdapter) GetVideo(url string) (*domain.VideoInfo, error) {
	baseArgs := a.getBaseArgs()
	fullArgs := append(baseArgs, "-J", "--no-playlist", url)

	cmd := exec.Command("/usr/bin/yt-dlp", fullArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp error: %s (%w)", string(output), err)
	}

	var data ytDlpJSON
	if err := json.Unmarshal(output, &data); err != nil {
		return nil, err
	}

	info := &domain.VideoInfo{
		ID:          data.ID,
		Title:       data.Title,
		Author:      data.Uploader,
		Description: data.Description,
		Duration:    fmt.Sprintf("%ds", int(data.Duration)),
		Thumbnail:   data.Thumbnail,
		Formats:     []domain.VideoFormat{},
	}

	for _, f := range data.Formats {
		if f.VCodec != "none" {
			label := f.FormatNote
			if label == "" {
				label = fmt.Sprintf("%dp", f.Height)
			}
			info.Formats = append(info.Formats, domain.VideoFormat{
				Label: label,
				Itag:  f.Height,
				Type:  f.Ext,
			})
		}
	}

	return info, nil
}

func (a *ytDlpAdapter) GetPlaylist(url string) (*domain.PlaylistInfo, error) {
	baseArgs := a.getBaseArgs()
	fullArgs := append(baseArgs, "-J", "--flat-playlist", url)

	cmd := exec.Command("/usr/bin/yt-dlp", fullArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp error: %s (%w)", string(output), err)
	}

	var data ytDlpJSON
	if err := json.Unmarshal(output, &data); err != nil {
		return nil, err
	}

	pl := &domain.PlaylistInfo{
		ID:     data.ID,
		Title:  data.Title,
		Author: data.Uploader,
		Videos: []domain.VideoInfo{},
	}

	for _, entry := range data.Entries {
		pl.Videos = append(pl.Videos, domain.VideoInfo{
			ID:       entry.ID,
			Title:    entry.Title,
			Author:   entry.Uploader,
			Duration: fmt.Sprintf("%ds", int(entry.Duration)),
		})
	}

	return pl, nil
}

func (a *ytDlpAdapter) Download(url, format, quality string) (string, error) {
	tempFile, err := os.CreateTemp("", "ytdlp-*.mp4")
	if err != nil {
		return "", err
	}
	tempFile.Close()
	os.Remove(tempFile.Name())

	baseArgs := a.getBaseArgs()
	args := append(baseArgs, "-o", tempFile.Name())

	if format == "mp3" {
		args = append(args, "-x", "--audio-format", "mp3")
	} else {
		args = append(args, "--merge-output-format", "mp4")
		if quality != "" {
			h := strings.TrimSuffix(quality, "p")
			fSelect := fmt.Sprintf("bestvideo[height<=%s]+bestaudio/best[height<=%s]", h, h)
			args = append(args, "-f", fSelect)
		} else {
			args = append(args, "-f", "bestvideo+bestaudio/best")
		}
	}

	args = append(args, url)

	cmd := exec.Command("/usr/bin/yt-dlp", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("download failed: %s, %w", string(out), err)
	}

	finalPath := tempFile.Name()
	if format == "mp3" && !strings.HasSuffix(finalPath, ".mp3") {
		finalPath += ".mp3"
	}

	return finalPath, nil
}
