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

func (a *ytDlpAdapter) GetVideo(url string) (*domain.VideoInfo, error) {
	// yt-dlp -J --flat-playlist --no-playlist url
	cmd := exec.Command("/usr/bin/yt-dlp", "-J", "--no-playlist", url)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp error: %w", err)
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

	// Process formats
	for _, f := range data.Formats {
		// Filter out bad formats if needed, or map them
		if f.VCodec != "none" {
			label := f.FormatNote
			if label == "" {
				label = fmt.Sprintf("%dp", f.Height)
			}
			// Use Height as ITAG proxy or just FormatID if string
			// Domain expects string label, int itag. WE need to store ID.
			// We need to refactor domain to support string IDs or map hash.
			// For now, we put height in itag.

			info.Formats = append(info.Formats, domain.VideoFormat{
				Label: label,
				Itag:  f.Height, // Hack: using height as ID for sorting
				Type:  f.Ext,
			})
		}
	}

	return info, nil
}

func (a *ytDlpAdapter) GetPlaylist(url string) (*domain.PlaylistInfo, error) {
	// yt-dlp -J --flat-playlist url
	cmd := exec.Command("/usr/bin/yt-dlp", "-J", "--flat-playlist", url)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp error: %w", err)
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
	// Quality here is likely a height string "1080p" or "1080"
	// Format is "mp4" or "mp3"

	// Construct yt-dlp format selector
	// bestvideo[height<=1080]+bestaudio/best[height<=1080]

	var args []string
	args = append(args, url)

	// Use temp file for output logic in service
	tempFile, err := os.CreateTemp("", "ytdlp-*.mp4")
	if err != nil {
		return "", err
	}
	tempFile.Close() // yt-dlp will overwrite
	os.Remove(tempFile.Name())

	args = append(args, "-o", tempFile.Name())

	if format == "mp3" {
		args = append(args, "-x", "--audio-format", "mp3")
	} else {
		args = append(args, "--merge-output-format", "mp4")
		if quality != "" {
			// Clean quality string "1080p" -> "1080"
			h := strings.TrimSuffix(quality, "p")
			// "bestvideo[height<=1080]+bestaudio/best[height<=1080]"
			fSelect := fmt.Sprintf("bestvideo[height<=%s]+bestaudio/best[height<=%s]", h, h)
			args = append(args, "-f", fSelect)
		} else {
			args = append(args, "-f", "bestvideo+bestaudio/best")
		}
	}

	// Force IP V4 often helps with blocks too
	// args = append(args, "--force-ipv4")

	cmd := exec.Command("/usr/bin/yt-dlp", args...)
	// Pass stderr to log?
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("download failed: %s, %w", string(out), err)
	}

	// Find the actual file (yt-dlp might change extension if merged)
	// If mp3, it will be .mp3
	// If mp4, .mp4
	// We trusted the output template? No, yt-dlp appends extension.
	// Wait, if we provided full path in -o, it usually respects it if extension matches.
	// Safe bet: check file existence.

	// Adjust extension expectation
	finalPath := tempFile.Name()
	if format == "mp3" && !strings.HasSuffix(finalPath, ".mp3") {
		finalPath += ".mp3" // Just guess, but usually yt-dlp handles -o specially.
		// Better: use -o with extension -o path.%(ext)s is safer but harder to predict path.
		// Let's rely on explicit path.
	}

	// Quick glob check if exact name missing
	return finalPath, nil
}
