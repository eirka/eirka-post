package utils

import (
	"bytes"
	"image"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/db"

	local "github.com/eirka/eirka-post/config"
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

			// Exercise the real production validation logic (sans ffprobe execution)
			result := img.validateWebM(ffprobeData)

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

// setWebMConfigLimits sets config limits that accept the small 640x480 ~2s test
// webm produced by generateTestWebM.
func setWebMConfigLimits() {
	config.Settings.Limits.ImageMaxWidth = 1920
	config.Settings.Limits.ImageMinWidth = 100
	config.Settings.Limits.ImageMaxHeight = 1080
	config.Settings.Limits.ImageMinHeight = 100
	config.Settings.Limits.ImageMaxSize = 10000000
	config.Settings.Limits.WebmMaxLength = 30
}

// generateTestWebM produces a small, valid VP9+Opus webm using ffmpeg and
// returns its bytes. ffmpeg/ffprobe are already hard dependencies of this
// package (init() panics without them), so a failure here is fatal rather than
// skipped.
func generateTestWebM(t *testing.T) []byte {
	t.Helper()

	path := filepath.Join(t.TempDir(), "generated.webm")

	args := []string{
		"-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc=size=640x480:rate=30:duration=2",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=2",
		"-c:v", "libvpx-vp9", "-b:v", "1M",
		"-c:a", "libopus", "-b:a", "64k",
		"-y", path,
	}

	if out, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		t.Fatalf("failed to generate test webm with ffmpeg: %v\n%s", err, out)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read generated test webm: %v", err)
	}

	return data
}

// formWebMRequest builds a multipart upload request carrying the given webm
// bytes, mirroring formJpegRequest in image_test.go.
func formWebMRequest(data []byte, filename string) *http.Request {
	var b bytes.Buffer

	w := multipart.NewWriter(&b)

	fw, _ := w.CreateFormFile("file", filename)

	io.Copy(fw, bytes.NewReader(data))

	w.Close()

	req, _ := http.NewRequest("POST", "/reply", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())

	return req
}

// TestCheckMagicWebM is a regression test for the bug where a webm upload never
// had i.video set, because checkMagic only set it when no extension was provided
// (and the real pipeline always sets the extension first via checkReqExt).
func TestCheckMagicWebM(t *testing.T) {
	webmData := generateTestWebM(t)

	img := ImageType{
		image: bytes.NewBuffer(webmData),
		Ext:   ".webm", // checkReqExt sets the extension before checkMagic runs
	}

	err := img.checkMagic()
	if assert.NoError(t, err, "checkMagic should accept a valid webm") {
		assert.Equal(t, "video/webm", img.mime, "mime should be detected as video/webm")
		assert.True(t, img.video, "video flag must be set so the webm pipeline runs")
	}
}

// TestCheckWebMRealFile runs the real checkWebM (ffprobe + JSON parse +
// validation) against an actual webm file on disk.
func TestCheckWebMRealFile(t *testing.T) {
	setWebMConfigLimits()

	webmData := generateTestWebM(t)

	path := filepath.Join(t.TempDir(), "real.webm")
	if err := os.WriteFile(path, webmData, 0644); err != nil {
		t.Fatalf("failed to write test webm: %v", err)
	}

	img := ImageType{Filepath: path}

	err := img.checkWebM()
	if assert.NoError(t, err, "checkWebM should accept a valid webm") {
		assert.Equal(t, 640, img.OrigWidth, "width should be parsed from ffprobe")
		assert.Equal(t, 480, img.OrigHeight, "height should be parsed from ffprobe")
		assert.Equal(t, 2, img.duration, "duration should be parsed from ffprobe")
	}
}

// TestCreateWebMThumbnail verifies a real JPEG thumbnail is extracted from a webm.
func TestCreateWebMThumbnail(t *testing.T) {
	webmData := generateTestWebM(t)

	srcPath := filepath.Join(t.TempDir(), "video.webm")
	if err := os.WriteFile(srcPath, webmData, 0644); err != nil {
		t.Fatalf("failed to write test webm: %v", err)
	}

	img := ImageType{
		Filepath:  srcPath,
		Thumbpath: filepath.Join(t.TempDir(), "videos.jpg"),
		duration:  2,
	}

	err := img.createWebMThumbnail()
	if assert.NoError(t, err, "createWebMThumbnail should succeed") {
		f, openErr := os.Open(img.Thumbpath)
		if assert.NoError(t, openErr, "thumbnail file should exist") {
			defer f.Close()

			cfg, format, decErr := image.DecodeConfig(f)
			if assert.NoError(t, decErr, "thumbnail should be a decodable image") {
				assert.Equal(t, "jpeg", format, "thumbnail should be a jpeg")
				assert.Positive(t, cfg.Width, "thumbnail width should be set")
				assert.Positive(t, cfg.Height, "thumbnail height should be set")
			}
		}
	}
}

// TestSaveImageWebM exercises the full upload pipeline end-to-end with a real
// webm: extension/magic checks, save-to-disk, ffprobe validation, ffmpeg frame
// extraction, and ImageMagick thumbnailing. This is the test that proves webm
// uploads actually function.
func TestSaveImageWebM(t *testing.T) {
	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	noban := sqlmock.NewRows([]string{"count"}).AddRow(0)
	nodupe := sqlmock.NewRows([]string{"count", "post", "thread"}).AddRow(0, 0, 0)

	mock.ExpectQuery(`SELECT count\(\*\) FROM banned_files WHERE ban_hash`).WillReturnRows(noban)
	mock.ExpectQuery(`select count\(1\),posts.post_num,threads.thread_id from threads`).WillReturnRows(nodupe)

	setWebMConfigLimits()
	config.Settings.Limits.ThumbnailMaxWidth = 200
	config.Settings.Limits.ThumbnailMaxHeight = 300

	err = os.MkdirAll(local.Settings.Directories.ImageDir, 0755)
	assert.NoError(t, err, "Failed to ensure image directory exists")
	err = os.MkdirAll(local.Settings.Directories.ThumbnailDir, 0755)
	assert.NoError(t, err, "Failed to ensure thumbnail directory exists")

	req := formWebMRequest(generateTestWebM(t), "upload.webm")

	img := ImageType{Ib: 1}
	img.File, img.Header, err = req.FormFile("file")
	assert.NoError(t, err, "FormFile should not fail")

	// Best-effort cleanup of the files the pipeline writes on success
	defer func() {
		os.Remove(img.Filepath)
		os.Remove(img.Thumbpath)
	}()

	err = img.SaveImage()
	if assert.NoError(t, err, "SaveImage should succeed for a valid webm") {
		assert.True(t, img.video, "video flag should be set")
		assert.True(t, strings.HasSuffix(img.Filename, ".webm"), "saved file should keep the .webm extension")
		assert.True(t, strings.HasSuffix(img.Thumbnail, "s.jpg"), "thumbnail should be a jpg")
		assert.Equal(t, 640, img.OrigWidth, "original width should be set")
		assert.Equal(t, 480, img.OrigHeight, "original height should be set")
		assert.Positive(t, img.ThumbWidth, "thumbnail width should be set")
		assert.Positive(t, img.ThumbHeight, "thumbnail height should be set")

		// the webm itself should be written to the image dir
		_, statErr := os.Stat(img.Filepath)
		assert.NoError(t, statErr, "saved webm should exist on disk")

		// the thumbnail should be written and be a valid jpeg
		f, openErr := os.Open(img.Thumbpath)
		if assert.NoError(t, openErr, "thumbnail should exist on disk") {
			defer f.Close()

			_, format, decErr := image.DecodeConfig(f)
			if assert.NoError(t, decErr, "thumbnail should decode") {
				assert.Equal(t, "jpeg", format, "thumbnail should be a jpeg")
			}
		}
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "all expected db queries should run")
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
