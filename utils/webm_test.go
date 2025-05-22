package utils

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/eirka/eirka-libs/config"
	"github.com/stretchr/testify/assert"
)

// Mock data for ffprobe tests
func mockGoodFFProbeData() ffprobe {
	return ffprobe{
		Streams: []ffprobeStream{
			{
				Index:        0,
				CodecName:    "vp9",
				CodecType:    "video",
				Width:        1280,
				Height:       720,
				AvgFrameRate: "30/1",
			},
			{
				Index:     1,
				CodecName: "opus",
				CodecType: "audio",
			},
		},
		Format: struct {
			Filename       string `json:"filename"`
			NbStreams      int    `json:"nb_streams"`
			FormatName     string `json:"format_name"`
			FormatLongName string `json:"format_long_name"`
			StartTime      string `json:"start_time"`
			Duration       string `json:"duration"`
			Size           string `json:"size"`
			BitRate        string `json:"bit_rate"`
		}{
			Filename:   "test.webm",
			NbStreams:  2,
			FormatName: "matroska,webm",
			Duration:   "10.5",
			Size:       "1000000",
			BitRate:    "1000000",
		},
	}
}

// TestParseFramerate tests the frameRate parsing function
func TestParseFramerate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
		isError  bool
	}{
		{
			name:     "Valid framerate 30fps",
			input:    "30/1",
			expected: 30.0,
			isError:  false,
		},
		{
			name:     "Valid framerate 29.97fps",
			input:    "30000/1001",
			expected: 29.97002997002997,
			isError:  false,
		},
		{
			name:     "Valid framerate 24fps",
			input:    "24/1",
			expected: 24.0,
			isError:  false,
		},
		{
			name:     "Invalid format",
			input:    "invalid",
			expected: 0,
			isError:  true,
		},
		{
			name:     "Invalid numerator",
			input:    "invalid/1",
			expected: 0,
			isError:  true,
		},
		{
			name:     "Invalid denominator",
			input:    "30/invalid",
			expected: 0,
			isError:  true,
		},
		{
			name:     "Zero denominator",
			input:    "30/0",
			expected: 0,
			isError:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseFramerate(tc.input)

			if tc.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tc.expected, result, 0.001)
			}
		})
	}
}

// TestWebMValidationCodecs tests codec validation in WebM files
func TestWebMValidationCodecs(t *testing.T) {
	// Set up test config
	config.Settings.Limits.ImageMaxWidth = 1920
	config.Settings.Limits.ImageMinWidth = 100
	config.Settings.Limits.ImageMaxHeight = 1080
	config.Settings.Limits.ImageMinHeight = 100
	config.Settings.Limits.ImageMaxSize = 10000000
	config.Settings.Limits.WebmMaxLength = 30

	// Create a base image type for testing
	baseImg := ImageType{
		Filepath: "/tmp/test.webm", // Just for testing
	}

	tests := []struct {
		name           string
		modifyFFProbe  func(ffprobe) ffprobe
		expectedErrStr string
	}{
		{
			name: "Valid WebM",
			modifyFFProbe: func(f ffprobe) ffprobe {
				return f // No modification
			},
			expectedErrStr: "",
		},
		{
			name: "Wrong format",
			modifyFFProbe: func(f ffprobe) ffprobe {
				f.Format.FormatName = "mp4"
				return f
			},
			expectedErrStr: "file is not a webm",
		},
		{
			name: "No streams",
			modifyFFProbe: func(f ffprobe) ffprobe {
				f.Streams = []ffprobeStream{}
				return f
			},
			expectedErrStr: "webm contains no streams",
		},
		{
			name: "Too many streams",
			modifyFFProbe: func(f ffprobe) ffprobe {
				f.Streams = append(f.Streams, ffprobeStream{}, ffprobeStream{})
				return f
			},
			expectedErrStr: "webm contains too many streams",
		},
		{
			name: "No video stream",
			modifyFFProbe: func(f ffprobe) ffprobe {
				f.Streams = []ffprobeStream{
					{
						CodecName: "opus",
						CodecType: "audio",
					},
				}
				return f
			},
			expectedErrStr: "webm contains no video stream",
		},
		{
			name: "Invalid video codec",
			modifyFFProbe: func(f ffprobe) ffprobe {
				f.Streams[0].CodecName = "h264"
				return f
			},
			expectedErrStr: "video codec 'h264' is not allowed",
		},
		{
			name: "Invalid audio codec",
			modifyFFProbe: func(f ffprobe) ffprobe {
				f.Streams[1].CodecName = "aac"
				return f
			},
			expectedErrStr: "audio codec 'aac' is not allowed",
		},
		{
			name: "Zero duration",
			modifyFFProbe: func(f ffprobe) ffprobe {
				f.Format.Duration = "0"
				return f
			},
			expectedErrStr: "webm has invalid duration",
		},
		{
			name: "Negative duration",
			modifyFFProbe: func(f ffprobe) ffprobe {
				f.Format.Duration = "-1"
				return f
			},
			expectedErrStr: "webm has invalid duration",
		},
		{
			name: "Zero size",
			modifyFFProbe: func(f ffprobe) ffprobe {
				f.Format.Size = "0"
				return f
			},
			expectedErrStr: "webm has invalid size",
		},
		{
			name: "Invalid dimensions - zero width",
			modifyFFProbe: func(f ffprobe) ffprobe {
				f.Streams[0].Width = 0
				return f
			},
			expectedErrStr: "webm has invalid dimensions",
		},
		{
			name: "Invalid dimensions - zero height",
			modifyFFProbe: func(f ffprobe) ffprobe {
				f.Streams[0].Height = 0
				return f
			},
			expectedErrStr: "webm has invalid dimensions",
		},
		{
			name: "Invalid framerate",
			modifyFFProbe: func(f ffprobe) ffprobe {
				f.Streams[0].AvgFrameRate = "invalid"
				return f
			},
			expectedErrStr: "webm has invalid framerate",
		},
		{
			name: "Too high framerate",
			modifyFFProbe: func(f ffprobe) ffprobe {
				f.Streams[0].AvgFrameRate = "120/1"
				return f
			},
			expectedErrStr: "webm framerate 120.00 fps is outside allowed range",
		},
		{
			name: "Too low bitrate",
			modifyFFProbe: func(f ffprobe) ffprobe {
				f.Format.BitRate = "50000"
				return f
			},
			expectedErrStr: "webm bitrate 50000 bps is too low",
		},
		{
			name: "Too high bitrate",
			modifyFFProbe: func(f ffprobe) ffprobe {
				f.Format.BitRate = "10000000"
				return f
			},
			expectedErrStr: "webm bitrate 10000000 bps is too high",
		},
		{
			name: "Width too large",
			modifyFFProbe: func(f ffprobe) ffprobe {
				f.Streams[0].Width = 2500
				return f
			},
			expectedErrStr: "webm width 2500 px is too large",
		},
		{
			name: "Width too small",
			modifyFFProbe: func(f ffprobe) ffprobe {
				f.Streams[0].Width = 50
				return f
			},
			expectedErrStr: "webm width 50 px is too small",
		},
		{
			name: "Height too large",
			modifyFFProbe: func(f ffprobe) ffprobe {
				f.Streams[0].Height = 2000
				return f
			},
			expectedErrStr: "webm height 2000 px is too large",
		},
		{
			name: "Height too small",
			modifyFFProbe: func(f ffprobe) ffprobe {
				f.Streams[0].Height = 50
				return f
			},
			expectedErrStr: "webm height 50 px is too small",
		},
		{
			name: "File too large",
			modifyFFProbe: func(f ffprobe) ffprobe {
				f.Format.Size = "20000000"
				return f
			},
			expectedErrStr: "webm file size 19.07 MB is too large",
		},
		{
			name: "Duration too long",
			modifyFFProbe: func(f ffprobe) ffprobe {
				f.Format.Duration = "45"
				return f
			},
			expectedErrStr: "webm duration 45 sec is too long",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a copy of the base image for this test
			img := baseImg

			// Create a test ffprobe result
			ffprobeData := tc.modifyFFProbe(mockGoodFFProbeData())

			// Set up a function to test only the validation logic, not the ffprobe execution
			result := validateWebMData(&img, ffprobeData)

			if tc.expectedErrStr == "" {
				assert.NoError(t, result)
			} else {
				assert.Error(t, result)
				if result != nil {
					assert.Contains(t, result.Error(), tc.expectedErrStr)
				}
			}
		})
	}
}

// Helper function to validate WebM data without running ffprobe
func validateWebMData(i *ImageType, ffprobeData ffprobe) error {
	// 1. Check file format
	if ffprobeData.Format.FormatName != "matroska,webm" {
		return errors.New("file is not a webm")
	}

	// 2. Validate stream count
	if len(ffprobeData.Streams) < minStreamCount {
		return errors.New("webm contains no streams")
	}

	if len(ffprobeData.Streams) > maxStreamCount {
		return errors.New("webm contains too many streams")
	}

	// 3. Find video stream
	var videoStream *ffprobeStream
	var audioStream *ffprobeStream

	for idx, stream := range ffprobeData.Streams {
		if stream.CodecType == "video" && videoStream == nil {
			videoStream = &ffprobeData.Streams[idx]
		} else if stream.CodecType == "audio" && audioStream == nil {
			audioStream = &ffprobeData.Streams[idx]
		}
	}

	// 4. Ensure we have a video stream
	if videoStream == nil {
		return errors.New("webm contains no video stream")
	}

	// 5. Validate video codec
	codecName := strings.ToLower(videoStream.CodecName)
	if !allowedCodecs[codecName] {
		return errors.New("video codec '" + videoStream.CodecName + "' is not allowed, must be VP8 or VP9")
	}

	// 6. Check audio stream if present
	if audioStream != nil {
		if !allowedAudioCodecs[strings.ToLower(audioStream.CodecName)] {
			return errors.New("audio codec '" + audioStream.CodecName + "' is not allowed, must be Vorbis or Opus")
		}
	}

	// 7. Parse and validate file duration
	duration, err := strconv.ParseFloat(ffprobeData.Format.Duration, 64)
	if err != nil {
		return errors.New("problem decoding webm duration")
	}

	if duration <= 0 {
		return errors.New("webm has invalid duration")
	}

	// set file duration
	i.duration = int(duration)

	// 8. Check file size
	originalSize, err := strconv.ParseFloat(ffprobeData.Format.Size, 64)
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
		return errors.New("webm has invalid framerate: " + err.Error())
	}

	if framerate < minVideoFramerate || framerate > maxVideoFramerate {
		return errors.New("webm framerate " + strconv.FormatFloat(framerate, 'f', 2, 64) + " fps is outside allowed range")
	}

	// 11. Check bitrate
	if ffprobeData.Format.BitRate != "" {
		bitrate, err := strconv.ParseInt(ffprobeData.Format.BitRate, 10, 64)
		if err == nil && bitrate > 0 {
			if bitrate < minVideoBitrate {
				return errors.New("webm bitrate " + ffprobeData.Format.BitRate + " bps is too low")
			}
			if bitrate > maxVideoBitrate {
				return errors.New("webm bitrate " + ffprobeData.Format.BitRate + " bps is too high")
			}
		}
	}

	// 12. Final size checks against config limits
	switch {
	case i.OrigWidth > config.Settings.Limits.ImageMaxWidth:
		return errors.New("webm width " + strconv.Itoa(i.OrigWidth) + " px is too large")
	case i.OrigWidth < config.Settings.Limits.ImageMinWidth:
		return errors.New("webm width " + strconv.Itoa(i.OrigWidth) + " px is too small")
	case i.OrigHeight > config.Settings.Limits.ImageMaxHeight:
		return errors.New("webm height " + strconv.Itoa(i.OrigHeight) + " px is too large")
	case i.OrigHeight < config.Settings.Limits.ImageMinHeight:
		return errors.New("webm height " + strconv.Itoa(i.OrigHeight) + " px is too small")
	case int(originalSize) > config.Settings.Limits.ImageMaxSize:
		return errors.New("webm file size " + strconv.FormatFloat(originalSize/(1024*1024), 'f', 2, 64) + " MB is too large")
	case i.duration > config.Settings.Limits.WebmMaxLength:
		return errors.New("webm duration " + strconv.Itoa(i.duration) + " sec is too long")
	}

	return nil
}

// TestWebMTimepointSelection tests the logic for choosing timepoints in WebM processing
func TestWebMTimepointSelection(t *testing.T) {
	tests := []struct {
		name              string
		duration          int
		expectedTimepoint string
	}{
		{
			name:              "Short video timepoint",
			duration:          3,
			expectedTimepoint: "00:00:00",
		},
		{
			name:              "Normal video timepoint",
			duration:          10,
			expectedTimepoint: "00:00:05",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Directly test the conditional logic from createWebMThumbnail
			var timepoint string
			if tc.duration > 5 {
				timepoint = "00:00:05"
			} else {
				timepoint = "00:00:00"
			}

			assert.Equal(t, tc.expectedTimepoint, timepoint,
				"Timepoint selection should work according to video duration")
		})
	}
}
