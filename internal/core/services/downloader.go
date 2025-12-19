package services

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"pagina/internal/core/domain"
	"pagina/internal/core/ports"
	"sort"
	"strconv"

	"github.com/kkdai/youtube/v2"
)

type downloaderService struct {
	ytRepo    ports.YouTubeRepository
	processor ports.MediaProcessor
}

func NewDownloaderService(yt ports.YouTubeRepository, proc ports.MediaProcessor) ports.DownloaderService {
	return &downloaderService{
		ytRepo:    yt,
		processor: proc,
	}
}

func (s *downloaderService) GetVideoInfo(url string) (*domain.VideoInfo, error) {
	video, err := s.ytRepo.GetVideo(url)
	if err != nil {
		return nil, err
	}

	info := &domain.VideoInfo{
		ID:          video.ID,
		Title:       video.Title,
		Author:      video.Author,
		Description: video.Description,
		Duration:    video.Duration.String(),
		Thumbnail:   getBestThumbnail(video.Thumbnails),
		Formats:     []domain.VideoFormat{},
	}

	seenQualities := make(map[string]bool)
	sort.Slice(video.Formats, func(i, j int) bool {
		return video.Formats[i].ItagNo > video.Formats[j].ItagNo
	})

	for _, f := range video.Formats {
		if f.MimeType == "" || f.QualityLabel == "" {
			continue
		}
		if _, exists := seenQualities[f.QualityLabel]; !exists {
			info.Formats = append(info.Formats, domain.VideoFormat{
				Label: f.QualityLabel,
				Itag:  f.ItagNo,
				Type:  f.MimeType,
			})
			seenQualities[f.QualityLabel] = true
		}
	}

	return info, nil
}

func (s *downloaderService) GetPlaylistInfo(urlInput string) (*domain.PlaylistInfo, error) {
	// Try extracting list ID if it's a watch URL with list param
	targetURL := urlInput
	u, err := url.Parse(urlInput)
	if err == nil {
		q := u.Query()
		if listID := q.Get("list"); listID != "" {
			targetURL = "https://www.youtube.com/playlist?list=" + listID
		}
	}

	playlist, err := s.ytRepo.GetPlaylist(targetURL)
	if err != nil {
		// If failed with manipulated URL (or original was already clean), try original just in case
		if targetURL != urlInput {
			playlist, err = s.ytRepo.GetPlaylist(urlInput)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	info := &domain.PlaylistInfo{
		ID:     playlist.ID,
		Title:  playlist.Title,
		Author: playlist.Author,
		Videos: []domain.VideoInfo{},
	}

	for _, v := range playlist.Videos {
		// Note: Playlist videos might have limited metadata compared to GetVideo
		info.Videos = append(info.Videos, domain.VideoInfo{
			ID:        v.ID,
			Title:     v.Title,
			Author:    v.Author,
			Duration:  v.Duration.String(),
			Thumbnail: getBestThumbnail(v.Thumbnails),
		})
	}

	return info, nil
}

func (s *downloaderService) DownloadVideo(url, formatStr, quality string) (io.ReadCloser, string, string, error) {
	video, err := s.ytRepo.GetVideo(url)
	if err != nil {
		return nil, "", "", err
	}

	if formatStr == "mp3" {
		return s.downloadMP3(video)
	}
	return s.downloadMP4(video, quality)
}

func (s *downloaderService) downloadMP4(video *youtube.Video, quality string) (io.ReadCloser, string, string, error) {
	var format *youtube.Format

	if quality != "" {
		// Try as ITAG first
		itag, err := strconv.Atoi(quality)
		if err == nil {
			formats := video.Formats.Itag(itag)
			if len(formats) > 0 {
				format = &formats[0]
			}
		} else {
			// Try as Quality Label (e.g. "1080p", "720p")
			var bestMatch *youtube.Format
			for _, f := range video.Formats {
				if f.QualityLabel == quality && f.AudioChannels > 0 { // Prefer with audio
					bestMatch = &f
					break
				}
				// Also try partial match or without audio (will mux later if needed)
				if f.QualityLabel == quality && bestMatch == nil {
					val := f // Copy loop var
					bestMatch = &val
				}
			}
			format = bestMatch
		}
	}

	// Fallback/Smart Selection
	if format == nil {
		formats := video.Formats.WithAudioChannels()
		for _, f := range formats {
			if f.AudioChannels > 0 {
				format = &f
				break
			}
		}
		if format == nil && len(video.Formats) > 0 {
			format = &video.Formats[0]
		}
	}

	if format == nil {
		return nil, "", "", fmt.Errorf("no suitable format found")
	}

	filename := fmt.Sprintf("%s.mp4", video.Title)
	contentType := "video/mp4"

	// Check muxing necessity
	if format.AudioChannels == 0 {
		return s.handleMuxing(video, format)
	}

	stream, _, err := s.ytRepo.GetStream(video, format)
	return stream, filename, contentType, err
}

func (s *downloaderService) handleMuxing(video *youtube.Video, videoFormat *youtube.Format) (io.ReadCloser, string, string, error) {
	audioFormats := video.Formats.Type("audio")
	if len(audioFormats) == 0 {
		return nil, "", "", fmt.Errorf("no audio streams found")
	}
	audioFormat := &audioFormats[0]

	// Download Video
	vStream, _, err := s.ytRepo.GetStream(video, videoFormat)
	if err != nil {
		return nil, "", "", err
	}
	defer vStream.Close()

	tempVideo, err := createTempFile("vid", vStream)
	if err != nil {
		return nil, "", "", err
	}
	defer os.Remove(tempVideo) // We will remove the source temp files, keep result

	// Download Audio
	aStream, _, err := s.ytRepo.GetStream(video, audioFormat)
	if err != nil {
		return nil, "", "", err
	}
	defer aStream.Close()

	tempAudio, err := createTempFile("aud", aStream)
	if err != nil {
		return nil, "", "", err
	}
	defer os.Remove(tempAudio)

	// Mux
	outputFile, err := s.processor.MuxVideoAudio(tempVideo, tempAudio)
	if err != nil {
		return nil, "", "", err
	}

	reader, _, _, err := newDeleteOnCloseReader(outputFile)
	if err != nil {
		return nil, "", "", err
	}
	// Override content type for MP4
	return reader, fmt.Sprintf("%s.mp4", video.Title), "video/mp4", nil
}

func (s *downloaderService) downloadMP3(video *youtube.Video) (io.ReadCloser, string, string, error) {
	formats := video.Formats.Type("audio")
	if len(formats) == 0 {
		// Fallback to any
		if len(video.Formats) > 0 {
			formats = video.Formats
		} else {
			return nil, "", "", fmt.Errorf("no formats found")
		}
	}
	format := &formats[0]

	stream, _, err := s.ytRepo.GetStream(video, format)
	if err != nil {
		return nil, "", "", err
	}
	defer stream.Close()

	tempRaw, err := createTempFile("raw", stream)
	if err != nil {
		return nil, "", "", err
	}
	defer os.Remove(tempRaw)

	outputFile, err := s.processor.ConvertToMP3(tempRaw)
	if err != nil {
		return nil, "", "", err
	}

	reader, _, _, err := newDeleteOnCloseReader(outputFile)
	if err != nil {
		return nil, "", "", err
	}

	filename := fmt.Sprintf("%s.mp3", video.Title)
	return reader, filename, "audio/mpeg", nil
}

// Helpers

func getBestThumbnail(thumbnails []youtube.Thumbnail) string {
	if len(thumbnails) == 0 {
		return ""
	}
	return thumbnails[len(thumbnails)-1].URL
}

func createTempFile(prefix string, r io.Reader) (string, error) {
	f, err := os.CreateTemp("", prefix+"-*")
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
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
	return &deleteOnCloseReader{File: f}, "", "application/octet-stream", nil // ContentType handled by caller mostly, but we can return generic here
}

func (d *deleteOnCloseReader) Close() error {
	path := d.File.Name()
	err := d.File.Close()
	_ = os.Remove(path) // Delete file on close
	return err
}
