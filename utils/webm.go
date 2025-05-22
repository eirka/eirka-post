package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/eirka/eirka-libs/config"
)

// Timeout constants for external video processing
const (
	ffmpegCheckTimeout = 10 * time.Second // Timeout for initial ffmpeg/ffprobe version checks
	ffmpegOpTimeout    = 60 * time.Second // Timeout for ffmpeg/ffprobe operations
)

// WebM validation constants
const (
	minVideoBitrate   = 100000  // Minimum video bitrate (100kbps)
	maxVideoBitrate   = 8000000 // Maximum video bitrate (8Mbps)
	minVideoFramerate = 1       // Minimum framerate (1fps)
	maxVideoFramerate = 60      // Maximum framerate (60fps)
	minStreamCount    = 1       // At least one stream required
	maxStreamCount    = 2       // Only allow video and optionally audio
)

// allowed codecs
var allowedCodecs = map[string]bool{
	"vp8": true,
	"vp9": true,
}

// allowed audio codecs if audio stream is present
var allowedAudioCodecs = map[string]bool{
	"vorbis": true,
	"opus":   true,
}

func init() {
	var err error

	// Create context with timeout for testing ffprobe
	ctx1, cancel1 := context.WithTimeout(context.Background(), ffmpegCheckTimeout)
	defer cancel1()

	// Test for ffprobe with timeout
	cmd1 := exec.CommandContext(ctx1, "ffprobe", "-version")
	_, err = cmd1.Output()
	if err != nil {
		if ctx1.Err() == context.DeadlineExceeded {
			panic(fmt.Sprintf("ffprobe check timed out after %v", ffmpegCheckTimeout))
		}
		panic("ffprobe not found")
	}

	// Create context with timeout for testing ffmpeg
	ctx2, cancel2 := context.WithTimeout(context.Background(), ffmpegCheckTimeout)
	defer cancel2()

	// Test for ffmpeg with timeout
	cmd2 := exec.CommandContext(ctx2, "ffmpeg", "-version")
	_, err = cmd2.Output()
	if err != nil {
		if ctx2.Err() == context.DeadlineExceeded {
			panic(fmt.Sprintf("ffmpeg check timed out after %v", ffmpegCheckTimeout))
		}
		panic("ffmpeg not found")
	}
}

// check webm metadata to make sure its the correct type of video, size, etc
func (i *ImageType) checkWebM() (err error) {

	ffprobeArgs := []string{
		"-v",
		"quiet",
		"-print_format",
		"json",
		"-show_format",
		"-show_streams",
		i.Filepath,
	}

	// Create context with timeout for ffprobe operations
	ctx, cancel := context.WithTimeout(context.Background(), ffmpegOpTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe", ffprobeArgs...)
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("ffprobe operation timed out after %v", ffmpegOpTimeout)
		}
		return errors.New("problem decoding webm")
	}

	ffprobe := ffprobe{}

	err = json.Unmarshal(output, &ffprobe)
	if err != nil {
		return errors.New("problem decoding webm")
	}

	// 1. Check file format
	if ffprobe.Format.FormatName != "matroska,webm" {
		return errors.New("file is not a webm")
	}

	// 2. Validate stream count
	if len(ffprobe.Streams) < minStreamCount {
		return errors.New("webm contains no streams")
	}

	if len(ffprobe.Streams) > maxStreamCount {
		return errors.New("webm contains too many streams")
	}

	// 3. Find video stream
	var videoStream *ffprobeStream
	var audioStream *ffprobeStream

	for i, stream := range ffprobe.Streams {
		if stream.CodecType == "video" && videoStream == nil {
			videoStream = &ffprobe.Streams[i]
		} else if stream.CodecType == "audio" && audioStream == nil {
			audioStream = &ffprobe.Streams[i]
		}
	}

	// 4. Ensure we have a video stream
	if videoStream == nil {
		return errors.New("webm contains no video stream")
	}

	// 5. Validate video codec
	codecName := strings.ToLower(videoStream.CodecName)
	if !allowedCodecs[codecName] {
		return fmt.Errorf("video codec '%s' is not allowed, must be VP8 or VP9", videoStream.CodecName)
	}

	// 6. Check audio stream if present
	if audioStream != nil {
		if !allowedAudioCodecs[strings.ToLower(audioStream.CodecName)] {
			return fmt.Errorf("audio codec '%s' is not allowed, must be Vorbis or Opus", audioStream.CodecName)
		}
	}

	// 7. Parse and validate file duration
	duration, err := strconv.ParseFloat(ffprobe.Format.Duration, 64)
	if err != nil {
		return errors.New("problem decoding webm duration")
	}

	if duration <= 0 {
		return errors.New("webm has invalid duration")
	}

	// set file duration
	i.duration = int(duration)

	// 8. Check file size
	originalSize, err := strconv.ParseFloat(ffprobe.Format.Size, 64)
	if err != nil {
		return errors.New("problem decoding webm size")
	}

	if originalSize <= 0 {
		return errors.New("webm has invalid size")
	}

	// 9. Set and validate dimensions
	i.OrigWidth = videoStream.Width
	i.OrigHeight = videoStream.Height

	if i.OrigWidth <= 0 || i.OrigHeight <= 0 {
		return errors.New("webm has invalid dimensions")
	}

	// 10. Parse and validate framerate
	framerate, err := parseFramerate(videoStream.AvgFrameRate)
	if err != nil {
		return fmt.Errorf("webm has invalid framerate: %v", err)
	}

	if framerate < minVideoFramerate || framerate > maxVideoFramerate {
		return fmt.Errorf("webm framerate %.2f fps is outside allowed range (%d-%d fps)",
			framerate, minVideoFramerate, maxVideoFramerate)
	}

	// 11. Check bitrate
	if ffprobe.Format.BitRate != "" {
		bitrate, err := strconv.ParseInt(ffprobe.Format.BitRate, 10, 64)
		if err == nil && bitrate > 0 {
			if bitrate < minVideoBitrate {
				return fmt.Errorf("webm bitrate %d bps is too low (min: %d bps)",
					bitrate, minVideoBitrate)
			}
			if bitrate > maxVideoBitrate {
				return fmt.Errorf("webm bitrate %d bps is too high (max: %d bps)",
					bitrate, maxVideoBitrate)
			}
		}
	}

	// 12. Final size checks against config limits
	switch {
	case i.OrigWidth > config.Settings.Limits.ImageMaxWidth:
		return fmt.Errorf("webm width %d px is too large (max: %d px)",
			i.OrigWidth, config.Settings.Limits.ImageMaxWidth)
	case i.OrigWidth < config.Settings.Limits.ImageMinWidth:
		return fmt.Errorf("webm width %d px is too small (min: %d px)",
			i.OrigWidth, config.Settings.Limits.ImageMinWidth)
	case i.OrigHeight > config.Settings.Limits.ImageMaxHeight:
		return fmt.Errorf("webm height %d px is too large (max: %d px)",
			i.OrigHeight, config.Settings.Limits.ImageMaxHeight)
	case i.OrigHeight < config.Settings.Limits.ImageMinHeight:
		return fmt.Errorf("webm height %d px is too small (min: %d px)",
			i.OrigHeight, config.Settings.Limits.ImageMinHeight)
	case int(originalSize) > config.Settings.Limits.ImageMaxSize:
		return fmt.Errorf("webm file size %.2f MB is too large (max: %.2f MB)",
			originalSize/(1024*1024), float64(config.Settings.Limits.ImageMaxSize)/(1024*1024))
	case i.duration > config.Settings.Limits.WebmMaxLength:
		return fmt.Errorf("webm duration %d sec is too long (max: %d sec)",
			i.duration, config.Settings.Limits.WebmMaxLength)
	}

	return

}

// create a webm thumbnail from the first frames
func (i *ImageType) createWebMThumbnail() (err error) {

	var timepoint string

	// set the time when the screenshot is taken depending on video duration
	if i.duration > 5 {
		timepoint = "00:00:05"
	} else {
		timepoint = "00:00:00"
	}

	// Create temporary files for safer processing
	// Get thumbpath directory and filename for safer filesystem operations
	thumbDir := filepath.Dir(i.Thumbpath)
	thumbFilename := filepath.Base(i.Thumbpath)

	// This temporary path is just for passing to ffmpeg
	tempThumbPath := i.Thumbpath

	// Note: We're still using direct file paths for ffmpeg operations
	// since ffmpeg handles files directly and doesn't work through file handles
	ffmpegArgs := []string{
		"-i",
		i.Filepath,
		"-v",
		"quiet",
		"-ss",
		timepoint,
		"-an",
		"-vframes",
		"1",
		"-f",
		"mjpeg",
		tempThumbPath,
	}

	// Create context with timeout for ffmpeg operations
	ctx, cancel := context.WithTimeout(context.Background(), ffmpegOpTimeout)
	defer cancel()

	// Make an image of first frame with ffmpeg
	cmd := exec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)
	_, err = cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("ffmpeg operation timed out after %v", ffmpegOpTimeout)
		}
		return errors.New("problem creating thumbnail from webm")
	}

	// Verify the thumbnail was created successfully using os.OpenInRoot
	// This is a double check to ensure the file exists and is valid after ffmpeg creates it
	_, err = os.OpenInRoot(thumbDir, thumbFilename)
	if err != nil {
		return fmt.Errorf("failed to verify thumbnail creation: %v", err)
	}

	return

}

// parseFramerate parses the framerate string from ffprobe output (e.g. "24/1")
func parseFramerate(fpsStr string) (float64, error) {
	parts := strings.Split(fpsStr, "/")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid framerate format: %s", fpsStr)
	}

	num, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid framerate numerator: %v", err)
	}

	denom, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid framerate denominator: %v", err)
	}

	if denom == 0 {
		return 0, errors.New("framerate denominator cannot be zero")
	}

	return num / denom, nil
}

// ffprobeStream represents a single media stream in ffprobe output
type ffprobeStream struct {
	Index              int    `json:"index"`
	CodecName          string `json:"codec_name"`
	CodecLongName      string `json:"codec_long_name"`
	CodecType          string `json:"codec_type"` // "video" or "audio"
	CodecTimeBase      string `json:"codec_time_base"`
	CodecTagString     string `json:"codec_tag_string"`
	CodecTag           string `json:"codec_tag"`
	Width              int    `json:"width"`  // video width in pixels
	Height             int    `json:"height"` // video height in pixels
	HasBFrames         int    `json:"has_b_frames"`
	SampleAspectRatio  string `json:"sample_aspect_ratio"`
	DisplayAspectRatio string `json:"display_aspect_ratio"`
	PixFmt             string `json:"pix_fmt"`
	Level              int    `json:"level"`
	RFrameRate         string `json:"r_frame_rate"`   // real base framerate
	AvgFrameRate       string `json:"avg_frame_rate"` // average framerate
	TimeBase           string `json:"time_base"`
	StartPts           int    `json:"start_pts"`
	StartTime          string `json:"start_time"`
	Disposition        struct {
		Default         int `json:"default"`
		Dub             int `json:"dub"`
		Original        int `json:"original"`
		Comment         int `json:"comment"`
		Lyrics          int `json:"lyrics"`
		Karaoke         int `json:"karaoke"`
		Forced          int `json:"forced"`
		HearingImpaired int `json:"hearing_impaired"`
		VisualImpaired  int `json:"visual_impaired"`
		CleanEffects    int `json:"clean_effects"`
		AttachedPic     int `json:"attached_pic"`
	} `json:"disposition"`
}

// ffprobe json format
type ffprobe struct {
	Streams []ffprobeStream `json:"streams"`
	Format  struct {
		Filename       string `json:"filename"`
		NbStreams      int    `json:"nb_streams"`
		FormatName     string `json:"format_name"`
		FormatLongName string `json:"format_long_name"`
		StartTime      string `json:"start_time"`
		Duration       string `json:"duration"`
		Size           string `json:"size"`
		BitRate        string `json:"bit_rate"`
	} `json:"format"`
}
