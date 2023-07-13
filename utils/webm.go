package utils

import (
	"encoding/json"
	"errors"
	"os/exec"
	"strconv"
	"strings"

	"github.com/eirka/eirka-libs/config"
)

// allowed codecs
var codecs = map[string]bool{
	"vp8": true,
	"vp9": true,
}

func init() {

	var err error

	// test for ffprobe
	_, err = exec.Command("ffprobe", "-version").Output()
	if err != nil {
		panic("ffprobe not found")
	}

	// test for ffmpeg
	_, err = exec.Command("ffmpeg", "-version").Output()
	if err != nil {
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

	cmd, err := exec.Command("ffprobe", ffprobeArgs...).Output()
	if err != nil {
		return errors.New("problem decoding webm")
	}

	ffprobe := ffprobe{}

	err = json.Unmarshal(cmd, &ffprobe)
	if err != nil {
		return errors.New("problem decoding webm")
	}

	switch {
	case ffprobe.Format.FormatName != "matroska,webm":
		return errors.New("file is not a webm")
	case !codecs[strings.ToLower(ffprobe.Streams[0].CodecName)]:
		return errors.New("file is not allowed webm codec")
	}

	duration, err := strconv.ParseFloat(ffprobe.Format.Duration, 64)
	if err != nil {
		return errors.New("problem decoding webm")
	}

	// set file duration
	i.duration = int(duration)

	originalSize, err := strconv.ParseFloat(ffprobe.Format.Size, 64)
	if err != nil {
		return
	}

	i.OrigWidth = ffprobe.Streams[0].Width
	i.OrigHeight = ffprobe.Streams[0].Height

	// Check against maximum sizes
	switch {
	case i.OrigWidth > config.Settings.Limits.ImageMaxWidth:
		return errors.New("webm width too large")
	case i.OrigWidth < config.Settings.Limits.ImageMinWidth:
		return errors.New("webm width too small")
	case i.OrigHeight > config.Settings.Limits.ImageMaxHeight:
		return errors.New("webm height too large")
	case i.OrigHeight < config.Settings.Limits.ImageMinHeight:
		return errors.New("webm height too small")
	case int(originalSize) > config.Settings.Limits.ImageMaxSize:
		return errors.New("webm size too large")
	case i.duration > config.Settings.Limits.WebmMaxLength:
		return errors.New("webm too long")
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
		i.Thumbpath,
	}

	// Make an image of first frame with ffmpeg
	_, err = exec.Command("ffmpeg", ffmpegArgs...).Output()
	if err != nil {
		return errors.New("problem decoding webm")
	}

	return

}

// ffprobe json format
type ffprobe struct {
	Streams []struct {
		Index              int    `json:"index"`
		CodecName          string `json:"codec_name"`
		CodecLongName      string `json:"codec_long_name"`
		CodecType          string `json:"codec_type"`
		CodecTimeBase      string `json:"codec_time_base"`
		CodecTagString     string `json:"codec_tag_string"`
		CodecTag           string `json:"codec_tag"`
		Width              int    `json:"width"`
		Height             int    `json:"height"`
		HasBFrames         int    `json:"has_b_frames"`
		SampleAspectRatio  string `json:"sample_aspect_ratio"`
		DisplayAspectRatio string `json:"display_aspect_ratio"`
		PixFmt             string `json:"pix_fmt"`
		Level              int    `json:"level"`
		RFrameRate         string `json:"r_frame_rate"`
		AvgFrameRate       string `json:"avg_frame_rate"`
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
	} `json:"streams"`
	Format struct {
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
