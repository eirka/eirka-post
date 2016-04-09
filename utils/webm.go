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

	// test for avprobe
	_, err = exec.Command("avprobe", "-version").Output()
	if err != nil {
		panic("avprobe not found")
	}

	// test for avconv
	_, err = exec.Command("avconv", "-version").Output()
	if err != nil {
		panic("avconv not found")
	}

}

// check webm metadata to make sure its the correct type of video, size, etc
func (i *ImageType) checkWebM() (err error) {

	avprobeArgs := []string{
		"-v",
		"quiet",
		"-of",
		"json",
		"-show_format",
		"-show_streams",
		i.Filepath,
	}

	cmd, err := exec.Command("avprobe", avprobeArgs...).Output()
	if err != nil {
		return errors.New("Problem decoding webm")
	}

	avprobe := avprobe{}

	err = json.Unmarshal(cmd, &avprobe)
	if err != nil {
		return errors.New("Problem decoding webm")
	}

	switch {
	case avprobe.Format.FormatName != "matroska,webm":
		return errors.New("File is not vp8 video")
	case !codecs[strings.ToLower(avprobe.Streams[0].CodecName)]:
		return errors.New("File is not allowed WebM codec")
	}

	duration, err := strconv.ParseFloat(avprobe.Format.Duration, 64)
	if err != nil {
		return errors.New("Problem decoding WebM")
	}

	// set file duration
	i.duration = int(duration)

	originalSize, err := strconv.ParseFloat(avprobe.Format.Size, 64)
	if err != nil {
		return
	}

	i.OrigWidth = avprobe.Streams[0].Width
	i.OrigHeight = avprobe.Streams[0].Height

	// Check against maximum sizes
	switch {
	case i.OrigWidth > config.Settings.Limits.ImageMaxWidth:
		return errors.New("WebM width too large")
	case i.OrigWidth < config.Settings.Limits.ImageMinWidth:
		return errors.New("WebM width too small")
	case i.OrigHeight > config.Settings.Limits.ImageMaxHeight:
		return errors.New("WebM height too large")
	case i.OrigHeight < config.Settings.Limits.ImageMinHeight:
		return errors.New("WebM height too small")
	case int(originalSize) > config.Settings.Limits.ImageMaxSize:
		return errors.New("WebM size too large")
	case i.duration > config.Settings.Limits.WebmMaxLength:
		return errors.New("WebM too long")
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

	avconvArgs := []string{
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

	// Make an image of first frame with avconv
	_, err = exec.Command("avconv", avconvArgs...).Output()
	if err != nil {
		return errors.New("Problem decoding WebM")
	}

	return

}

// avprobe json format
type avprobe struct {
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
		AvgFrameRate       string `json:"avg_frame_rate"`
		TimeBase           string `json:"time_base"`
		StartTime          string `json:"start_time"`
		Duration           string `json:"duration"`
	} `json:"streams"`
}
