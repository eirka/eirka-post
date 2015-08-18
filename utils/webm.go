package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/techjanitor/pram-post/config"
)

func (i *ImageType) SaveWebM() (err error) {

	// save the file
	err = i.saveFile()
	if err != nil {
		return
	}

	// check the webm info
	err = i.checkWebM()
	if err != nil {
		return
	}

	// create thumbnail from webm
	err = i.createWebMThumbnail()
	if err != nil {
		return
	}

	return

}

// check webm metadata to make sure its the correct type of video, size, etc
func (i *ImageType) checkWebM() (err error) {
	imagefile := filepath.Join(config.Settings.General.ImageDir, i.Filename)

	avprobeArgs := []string{
		"-v",
		"quiet",
		"-of",
		"json",
		"-show_format",
		"-show_streams",
		imagefile,
	}

	cmd, err := exec.Command("avprobe", avprobeArgs...).Output()
	if err != nil {
		return errors.New("problem decoding webm")
	}

	avprobe := avprobe{}

	err = json.Unmarshal(cmd, &avprobe)
	if err != nil {
		os.RemoveAll(imagefile)
		return errors.New("problem decoding webm")
	}

	// allowed codecs
	codecs := map[string]bool{
		"vp8": true,
		"vp9": true,
	}

	switch {
	case avprobe.Format.FormatName != "matroska,webm":
		os.RemoveAll(imagefile)
		return errors.New("file is not vp8 video")
	case !codecs[strings.ToLower(avprobe.Streams[0].CodecName)]:
		os.RemoveAll(imagefile)
		return errors.New("file is not allowed webm codec")
	}

	duration, err := strconv.ParseFloat(avprobe.Format.Duration, 64)
	if err != nil {
		os.RemoveAll(imagefile)
		return errors.New("problem decoding webm")
	}

	// set file duration
	i.duration = int(duration)

	orig_size, err := strconv.ParseFloat(avprobe.Format.Size, 64)
	if err != nil {
		os.RemoveAll(imagefile)
		return
	}

	i.OrigWidth = avprobe.Streams[0].Width
	i.OrigHeight = avprobe.Streams[0].Height

	// Check against maximum sizes
	switch {
	case i.OrigWidth > config.Settings.Limits.ImageMaxWidth:
		os.RemoveAll(imagefile)
		return errors.New("webm width too large")
	case i.OrigWidth < config.Settings.Limits.ImageMinWidth:
		os.RemoveAll(imagefile)
		return errors.New("webm width too small")
	case i.OrigHeight > config.Settings.Limits.ImageMaxHeight:
		os.RemoveAll(imagefile)
		return errors.New("webm height too large")
	case i.OrigHeight < config.Settings.Limits.ImageMinHeight:
		os.RemoveAll(imagefile)
		return errors.New("webm height too small")
	case int(orig_size) > config.Settings.Limits.ImageMaxSize:
		os.RemoveAll(imagefile)
		return errors.New("webm size too large")
	case i.duration > config.Settings.Limits.WebmMaxLength:
		os.RemoveAll(imagefile)
		return errors.New("webm too long")
	}

	return

}

// create a webm thumbnail from the first frames
func (i *ImageType) createWebMThumbnail() (err error) {
	imagefile := filepath.Join(config.Settings.General.ImageDir, i.Filename)
	thumbfile := filepath.Join(config.Settings.General.ThumbnailDir, i.Thumbnail)

	var timepoint string

	// set the time when the screenshot is taken depending on video duration
	if i.duration > 5 {
		timepoint = "00:00:05"
	} else {
		timepoint = "00:00:00"
	}

	avconvArgs := []string{
		"-i",
		imagefile,
		"-v",
		"quiet",
		"-ss",
		timepoint,
		"-an",
		"-vframes",
		"1",
		"-f",
		"mjpeg",
		thumbfile,
	}

	// Make an image of first frame with avconv
	_, err = exec.Command("avconv", avconvArgs...).Output()
	if err != nil {
		os.RemoveAll(thumbfile)
		os.RemoveAll(imagefile)
		return errors.New("problem decoding webm")
	}

	orig_dimensions := fmt.Sprintf("%dx%d", i.OrigWidth, i.OrigHeight)
	thumb_dimensions := fmt.Sprintf("%dx%d>", config.Settings.Limits.ThumbnailMaxWidth, config.Settings.Limits.ThumbnailMaxHeight)
	imagef := fmt.Sprintf("%s[0]", thumbfile)

	args := []string{
		"-background",
		"white",
		"-flatten",
		"-size",
		orig_dimensions,
		"-resize",
		thumb_dimensions,
		"-quality",
		"90",
		imagef,
		thumbfile,
	}

	_, err = exec.Command("convert", args...).Output()
	if err != nil {
		os.RemoveAll(thumbfile)
		os.RemoveAll(imagefile)
		return errors.New("problem making thumbnail")
	}

	thumb, err := os.Open(thumbfile)
	if err != nil {
		os.RemoveAll(thumbfile)
		os.RemoveAll(imagefile)
		return errors.New("problem making thumbnail")
	}
	defer thumb.Close()

	img, _, err := image.DecodeConfig(thumb)
	if err != nil {
		os.RemoveAll(thumbfile)
		os.RemoveAll(imagefile)
		return errors.New("problem decoding thumbnail")
	}

	i.ThumbWidth = img.Width
	i.ThumbHeight = img.Height

	err = UploadGCS(thumbfile, fmt.Sprintf("thumb/%s", i.Thumbnail))
	if err != nil {
		os.RemoveAll(thumbfile)
		os.RemoveAll(imagefile)
		return
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
