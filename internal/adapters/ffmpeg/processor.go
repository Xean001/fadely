package ffmpeg

import (
	"log"
	"os/exec"
	"pagina/internal/core/ports"
)

type ffmpegProcessor struct{}

func NewFFmpegProcessor() ports.MediaProcessor {
	return &ffmpegProcessor{}
}

func (p *ffmpegProcessor) MuxVideoAudio(videoFile, audioFile string) (string, error) {
	outputFile := videoFile + "-muxed.mp4"

	// Force H.264 generic for max compatibility (re-encoding needed for VP9/WebM sources)
	// -pix_fmt yuv420p is often needed for compatibility
	cmd := exec.Command("ffmpeg", "-y", "-i", videoFile, "-i", audioFile, "-c:v", "libx264", "-preset", "fast", "-crf", "23", "-pix_fmt", "yuv420p", "-c:a", "aac", "-b:a", "128k", "-movflags", "+faststart", outputFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("FFmpeg mux/encode error: %v, output: %s", err, string(out))
		return "", err
	}

	return outputFile, nil
}

func (p *ffmpegProcessor) ConvertToMP3(audioFile string) (string, error) {
	outputFile := audioFile + ".mp3"

	cmd := exec.Command("ffmpeg", "-y", "-i", audioFile, "-f", "mp3", "-ab", "192k", "-vn", outputFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("FFmpeg convert error: %v, output: %s", err, string(out))
		return "", err
	}

	return outputFile, nil
}
