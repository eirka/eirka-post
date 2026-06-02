package utils

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"slices"

	// gif support
	_ "image/gif"
	// jpeg support
	_ "image/jpeg"
	// png support
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	shortid "github.com/teris-io/shortid"

	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/db"

	local "github.com/eirka/eirka-post/config"
)

// Timeout constants for external processes
const (
	initialCheckTimeout = 10 * time.Second // Timeout for initial version checks
	processTimeout      = 60 * time.Second // Timeout for image processing operations
)

// maxImagePixels caps the total decoded pixel area (width*height) of an image,
// independent of the per-dimension limits, to bound ImageMagick memory use and
// reject decompression bombs (a tiny, highly-compressed file declaring enormous
// dimensions). It is a var (not const) so tests can lower it without having to
// synthesize a real 100-megapixel image.
var maxImagePixels int64 = 100 * 1000 * 1000 // 100 megapixels

// magickLimits caps ImageMagick resource consumption per invocation. The
// context timeout only bounds wall-clock, not memory, so these protect against
// memory/disk exhaustion from a crafted or oversized image.
var magickLimits = []string{
	"-limit", "memory", "256MiB",
	"-limit", "map", "512MiB",
	"-limit", "area", "256MB",
	"-limit", "disk", "1GiB",
}

// magickCoder maps a validated file extension to an explicit ImageMagick coder
// so convert decodes the input as its verified type and cannot be coerced into a
// different (e.g. delegate-backed) coder by crafted file content.
func magickCoder(ext string) string {
	switch ext {
	case ".png":
		return "png"
	case ".gif":
		return "gif"
	default: // .jpg / .jpeg
		return "jpg"
	}
}

// valid file extensions with their corresponding MIME types
var validExtAndMime = map[string][]string{
	".jpg":  {"image/jpeg"},
	".jpeg": {"image/jpeg"},
	".png":  {"image/png"},
	".gif":  {"image/gif"},
	".webm": {"video/webm"},
}

// valid file extensions
var validExt = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webm": true,
}

func init() {
	var err error

	// Create context with timeout for testing ImageMagick
	ctx, cancel := context.WithTimeout(context.Background(), initialCheckTimeout)
	defer cancel()

	// Test for ImageMagick
	cmd := exec.CommandContext(ctx, "convert", "--version")
	_, err = cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			panic(fmt.Sprintf("ImageMagick check timed out after %v", initialCheckTimeout))
		}
		panic("ImageMagick not found")
	}
}

// FileUploader defines the file processing functions
type FileUploader interface {
	// struct integrity
	IsValid() bool
	IsValidPost() bool

	// image processing
	SaveImage() (err error)
	checkReqExt() (err error)
	copyBytes() (err error)
	getHash() (err error)
	checkBanned() (err error)
	checkDuplicate() (err error)
	checkMagic() (err error)
	getStats() (err error)
	saveFile() (err error)
	makeFilenames() (err error)
	createThumbnail(maxwidth, maxheight int) (err error)
	cleanupFiles() // cleanup files on error

	// webm specific functions
	checkWebM() (err error)
	createWebMThumbnail() (err error)

	// avatar functions
	SaveAvatar() (err error)
}

// ImageType defines an image and its metadata for processing
type ImageType struct {
	File        multipart.File
	Header      *multipart.FileHeader
	Ib          uint
	Filename    string
	Thumbnail   string
	Filepath    string
	Thumbpath   string
	Ext         string
	MD5         string
	SHA         string
	OrigWidth   int
	OrigHeight  int
	ThumbWidth  int
	ThumbHeight int
	image       *bytes.Buffer
	mime        string
	duration    int
	video       bool
	avatar      bool
}

var _ = FileUploader(&ImageType{})

// IsValid will check struct integrity
func (i *ImageType) IsValid() bool {

	if i.Filename == "" {
		return false
	}

	if i.Filepath == "" {
		return false
	}

	if i.Thumbnail == "" {
		return false
	}

	if i.Thumbpath == "" {
		return false
	}

	if i.Ib == 0 {
		return false
	}

	if i.Ext == "" {
		return false
	}

	if i.MD5 == "" {
		return false
	}

	if i.SHA == "" {
		return false
	}

	if i.mime == "" {
		return false
	}

	return true
}

// IsValidPost will check final struct integrity
func (i *ImageType) IsValidPost() bool {
	if i.OrigWidth == 0 {
		return false
	}

	if i.OrigHeight == 0 {
		return false
	}

	if i.ThumbWidth == 0 {
		return false
	}

	if i.ThumbHeight == 0 {
		return false
	}

	return true
}

// SaveImage runs the entire file processing pipeline
func (i *ImageType) SaveImage() (err error) {
	// Successful completion flag
	var success bool
	// Defer cleanup on failure
	defer func() {
		// If we didn't complete successfully, clean up any created files
		if !success {
			i.cleanupFiles()
		}
	}()

	// check given file ext
	err = i.checkReqExt()
	if err != nil {
		return
	}

	// copy the multipart file into a buffer
	err = i.copyBytes()
	if err != nil {
		return
	}

	// get file md5
	err = i.getHash()
	if err != nil {
		return
	}

	// check to see if the file is banned
	err = i.checkBanned()
	if err != nil {
		return
	}

	// check to see if the file already exists
	err = i.checkDuplicate()
	if err != nil {
		return
	}

	// check file magic sig
	err = i.checkMagic()
	if err != nil {
		return
	}

	// check image stats
	err = i.getStats()
	if err != nil {
		return
	}

	// save the file to disk
	err = i.saveFile()
	if err != nil {
		return
	}

	// process a webm
	if i.video {
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
	}

	// create a thumbnail
	err = i.createThumbnail(config.Settings.Limits.ThumbnailMaxWidth, config.Settings.Limits.ThumbnailMaxHeight)
	if err != nil {
		return
	}

	// check final state
	if !i.IsValidPost() {
		err = errors.New("ImageType is not valid")
		return
	}

	// Mark as successful to prevent cleanup
	success = true
	return
}

// Get file extension from request header and perform security checks
func (i *ImageType) checkReqExt() (err error) {
	// Get ext from request header
	if i.Header == nil {
		return errors.New("no file header provided")
	}

	// Cheap pre-read gate: reject an upload whose declared size already exceeds
	// the configured maximum, before it is buffered/hashed/written. Header.Size
	// is set by the multipart parser but is client-influenced, so copyBytes
	// still enforces the real bound while reading.
	if max := config.Settings.Limits.ImageMaxSize; max > 0 && i.Header.Size > int64(max) {
		return errors.New("image size is too large")
	}

	name := i.Header.Filename
	if name == "" {
		return errors.New("no filename provided")
	}

	// Extract the extension
	ext := filepath.Ext(name)
	if ext == "" {
		return errors.New("no file extension")
	}

	// Convert to lowercase for consistent checking
	ext = strings.ToLower(ext)

	// Check if the filename contains multiple extensions
	// This prevents attacks like example.php.jpg
	if strings.Count(name, ".") > 1 {
		// Additional check: only allow if the penultimate extension is safe
		parts := strings.Split(name, ".")
		if len(parts) > 2 {
			for _, dangerousExt := range []string{"php", "exe", "js", "html", "htm", "bat", "sh", "cgi", "pl", "asp", "aspx", "py", "rb"} {
				// Check if any suspicious extension appears before the final extension
				for i := 0; i < len(parts)-1; i++ {
					if strings.ToLower(parts[i]) == dangerousExt {
						return errors.New("suspicious file extension pattern detected")
					}
				}
			}
		}
	}

	// Check to see if extension is allowed
	if !isAllowedExt(ext) {
		return errors.New("format not supported")
	}

	i.Ext = ext
	return
}

// Check if file ext allowed
func isAllowedExt(ext string) bool {
	ext = strings.ToLower(ext)
	// Check for double extensions (e.g., .php.jpg)
	if strings.Count(ext, ".") > 1 {
		return false
	}
	return validExt[ext]
}

// Copy the multipart file into a bytes buffer
func (i *ImageType) copyBytes() (err error) {
	defer i.File.Close()

	i.image = new(bytes.Buffer)

	// Bound the copy by the configured maximum image size so a malicious or
	// malformed upload cannot be buffered into memory without limit. We read at
	// most ImageMaxSize+1 bytes: if we actually get that many the file is over
	// the limit and is rejected here, before it is hashed, written to disk, or
	// (for webm) handed to ffprobe. A zero/negative limit is treated as "no
	// configured limit" to preserve availability on a misconfiguration (the
	// LimitBody middleware is the request-level backstop).
	max := int64(config.Settings.Limits.ImageMaxSize)
	if max <= 0 {
		if _, err = io.Copy(i.image, i.File); err != nil {
			return errors.New("problem copying file to buffer")
		}
		return nil
	}

	n, err := io.CopyN(i.image, i.File, max+1)
	if err != nil && err != io.EOF {
		return errors.New("problem copying file to buffer")
	}
	if n > max {
		return errors.New("image size is too large")
	}

	return nil
}

// Get image MD5 and write file into buffer
func (i *ImageType) getHash() (err error) {

	hasher := md5.New()

	sha := sha1.New()

	// read into hasher for md5
	_, err = io.Copy(hasher, bytes.NewReader(i.image.Bytes()))
	if err != nil {
		return errors.New("problem creating MD5 hash")
	}

	// Set md5sum from hasher
	i.MD5 = hex.EncodeToString(hasher.Sum(nil))

	// read into hasher for md5
	_, err = io.Copy(sha, bytes.NewReader(i.image.Bytes()))
	if err != nil {
		return errors.New("problem creating SHA1 hash")
	}

	// Set sha1
	i.SHA = hex.EncodeToString(sha.Sum(nil))

	return

}

// check if the md5 is a banned file
func (i *ImageType) checkBanned() (err error) {

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	if i.MD5 == "" {
		return errors.New("no hash set on file banned check")
	}

	var check bool

	err = dbase.QueryRow(`SELECT count(*) FROM banned_files WHERE ban_hash = ?`, i.MD5).Scan(&check)
	if err != nil {
		return
	}

	// return error if it exists
	if check {
		return fmt.Errorf("file is banned")
	}

	return
}

// check if the md5 is already in the database
func (i *ImageType) checkDuplicate() (err error) {

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	if i.MD5 == "" {
		return errors.New("no hash set on duplicate check")
	}

	if i.Ib == 0 {
		return errors.New("no imageboard set on duplicate check")
	}

	var check bool
	var thread, post sql.NullInt64

	err = dbase.QueryRow(`select count(1),posts.post_num,threads.thread_id from threads
	LEFT JOIN posts on threads.thread_id = posts.thread_id
	LEFT JOIN images on posts.post_id = images.post_id
	WHERE image_hash = ? AND ib_id = ? AND post_deleted = 0`, i.MD5, i.Ib).Scan(&check, &post, &thread)
	if err != nil {
		return
	}

	// return error if it exists
	if check {
		return fmt.Errorf("image has already been posted. Thread: %d Post: %d", thread.Int64, post.Int64)
	}

	return
}

func (i *ImageType) checkMagic() (err error) {
	if i.image == nil || i.image.Len() == 0 {
		return errors.New("no image data to analyze")
	}

	// Get the file's content for analysis
	fileBytes := i.image.Bytes()

	// Detect the MIME type from file content signatures
	i.mime = http.DetectContentType(fileBytes)

	// If an extension was provided earlier, verify it matches the detected type
	if i.Ext != "" {
		validMimeTypes, exists := validExtAndMime[i.Ext]
		if !exists {
			return errors.New("unsupported file extension")
		}

		validMime := slices.Contains(validMimeTypes, i.mime)

		if !validMime {
			return errors.New("file extension doesn't match content type")
		}
	} else {
		// Set extension based on detected MIME type if not provided
		switch i.mime {
		case "image/png":
			i.Ext = ".png"
		case "image/jpeg":
			i.Ext = ".jpg"
		case "image/gif":
			i.Ext = ".gif"
		case "video/webm":
			i.Ext = ".webm"
			// i.video is set in the signature-validation switch below
		default:
			return errors.New("unknown or unsupported file type")
		}
	}

	// Perform additional validation based on file type
	switch i.mime {
	case "image/png":
		// Validate PNG signature (first 8 bytes)
		if len(fileBytes) < 8 || string(fileBytes[0:8]) != "\x89PNG\r\n\x1a\n" {
			return errors.New("invalid PNG file signature")
		}
	case "image/jpeg":
		// JPEG files start with FF D8 and end with FF D9
		if len(fileBytes) < 2 || fileBytes[0] != 0xFF || fileBytes[1] != 0xD8 {
			return errors.New("invalid JPEG file signature")
		}
		// Require the EOI (end-of-image) marker as the final two bytes. We store
		// the original bytes verbatim, so rejecting files with data appended
		// after the JPEG stream closes a common way to smuggle a payload past a
		// magic-byte-only check. NOTE: this only catches *trailing* data; it does
		// not detect a payload embedded before EOI or between JPEG segments.
		// Fully neutralizing embedded payloads would require re-encoding the
		// image, which we intentionally do not do here.
		if !(fileBytes[len(fileBytes)-2] == 0xFF && fileBytes[len(fileBytes)-1] == 0xD9) {
			return errors.New("invalid JPEG: data after end-of-image marker")
		}
	case "image/gif":
		// Check GIF header (GIF87a or GIF89a)
		if len(fileBytes) < 6 {
			return errors.New("invalid GIF file (too small)")
		}
		header := string(fileBytes[0:6])
		if header != "GIF87a" && header != "GIF89a" {
			return errors.New("invalid GIF file signature")
		}
	case "video/webm":
		// Basic WebM check - validate EBML header
		// WebM files start with an EBML header (0x1A 0x45 0xDF 0xA3)
		if len(fileBytes) < 4 || fileBytes[0] != 0x1A || fileBytes[1] != 0x45 || fileBytes[2] != 0xDF || fileBytes[3] != 0xA3 {
			return errors.New("invalid WebM file signature")
		}
		// Flag as a video so the pipeline runs the webm-specific path. This must
		// happen here (not only in the extension-detection branch above) because
		// the upload pipeline always sets the extension first via checkReqExt.
		i.video = true
	}

	// Check for suspiciously small files that might be trying to bypass checks.
	// This applies to video too: a few-byte EBML stub would otherwise be written
	// to disk before ffprobe rejects it, and no real webm is anywhere near this
	// small.
	if len(fileBytes) < 100 {
		return errors.New("file is suspiciously small")
	}

	return nil
}

func (i *ImageType) getStats() (err error) {

	// skip if its a video since we cant decode it
	if i.video {
		return
	}

	// decode image config
	img, _, err := image.DecodeConfig(bytes.NewReader(i.image.Bytes()))
	if err != nil {
		return errors.New("problem decoding image")
	}

	// set original width
	i.OrigWidth = img.Width
	// set original height
	i.OrigHeight = img.Height

	// Guard against decompression bombs: a small, highly-compressed file can
	// declare a pixel area within the per-dimension limits that still forces a
	// huge allocation when ImageMagick decodes it. Reject on total pixel area
	// before any convert invocation.
	if int64(i.OrigWidth)*int64(i.OrigHeight) > maxImagePixels {
		return fmt.Errorf("image area too large. Max: %d megapixels", maxImagePixels/(1000*1000))
	}

	// Check against maximum sizes
	switch {
	case i.OrigWidth > config.Settings.Limits.ImageMaxWidth:
		return fmt.Errorf("image width too large. Max: %dpx", config.Settings.Limits.ImageMaxWidth)
	case img.Width < config.Settings.Limits.ImageMinWidth:
		return fmt.Errorf("image width too small. Min: %dpx", config.Settings.Limits.ImageMinWidth)
	case i.OrigHeight > config.Settings.Limits.ImageMaxHeight:
		return fmt.Errorf("image height too large. Max: %dpx", config.Settings.Limits.ImageMaxHeight)
	case img.Height < config.Settings.Limits.ImageMinHeight:
		return fmt.Errorf("image height too small. Min: %dpx", config.Settings.Limits.ImageMinHeight)
	case i.image.Len() > config.Settings.Limits.ImageMaxSize:
		return fmt.Errorf("image filesize too large. Max: %dMB", (config.Settings.Limits.ImageMaxSize/1024)/1024)
	}

	return

}

// setFilenames generates the on-disk filenames and applies the avatar-specific
// thumbnail override. Split out so saveFile can regenerate names on a collision.
func (i *ImageType) setFilenames() (err error) {
	// Generate filenames and paths
	if err = i.makeFilenames(); err != nil {
		return err
	}

	// avatar filename is the users id
	if i.avatar {
		i.Thumbnail = fmt.Sprintf("%d.png", i.Ib)
		i.Thumbpath = filepath.Join(local.Settings.Directories.AvatarDir, i.Thumbnail)
	}

	return nil
}

func (i *ImageType) saveFile() (err error) {
	// Reset the image buffer when done regardless of success or failure
	defer i.image.Reset()

	// Generate filenames and paths
	if err = i.setFilenames(); err != nil {
		return err
	}

	// Ensure we have valid data before attempting to save
	if !i.IsValid() {
		return errors.New("ImageType is not valid")
	}

	// Open the image dir with traversal-resistant root
	root, err := os.OpenRoot(local.Settings.Directories.ImageDir)
	if err != nil {
		return fmt.Errorf("failed to open directory: %v", err)
	}
	defer root.Close()

	// Create the file with O_EXCL so a filename collision can never silently
	// truncate/overwrite another post's image. On the (now very rare) collision,
	// regenerate the name and retry instead of clobbering existing data.
	var image *os.File
	for range 5 {
		image, err = root.OpenFile(i.Filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("problem creating file: %v", err)
		}
		// collision: regenerate the filename (and dependent paths) and retry
		if err = i.setFilenames(); err != nil {
			return err
		}
	}
	if err != nil {
		return fmt.Errorf("problem creating unique file: %v", err)
	}
	defer image.Close()

	// Write the image data to the file
	_, err = io.Copy(image, bytes.NewReader(i.image.Bytes()))
	if err != nil {
		return fmt.Errorf("problem writing to file: %v", err)
	}

	return nil
}

// nameGen is the single, process-wide shortid generator used for on-disk
// filenames. shortid's zero-collision guarantee depends on reusing ONE
// generator: it keeps mutex-protected last-millisecond and in-millisecond
// counter state and only appends the disambiguating counter when two ids are
// requested within the same millisecond. The previous code constructed a fresh
// generator on every upload, which reset that state each call, so two uploads
// landing in the same millisecond could produce identical filenames (and then
// silently overwrite each other on disk). Constructing it once here restores the
// guarantee. MustNew is safe at init: it only errors on an invalid worker or
// alphabet, both compile-time constants here.
var nameGen = shortid.MustNew(1, shortid.DefaultABC, 9001)

// Make a random unix time filename
func (i *ImageType) makeFilenames() (err error) {

	// generate filename from the shared generator (error-returning, so a
	// generation failure surfaces as a normal upload error instead of a panic)
	filename, err := nameGen.Generate()
	if err != nil {
		return fmt.Errorf("problem generating filename: %v", err)
	}

	// Append ext to filename
	i.Filename = fmt.Sprintf("%s%s", filename, i.Ext)

	// Append jpg to thumbnail name because it is always a jpg
	i.Thumbnail = fmt.Sprintf("%ss.jpg", filename)

	// set the full file path
	i.Filepath = filepath.Join(local.Settings.Directories.ImageDir, i.Filename)

	// set the full thumbnail path
	i.Thumbpath = filepath.Join(local.Settings.Directories.ThumbnailDir, i.Thumbnail)

	return nil
}

func (i *ImageType) createThumbnail(maxwidth, maxheight int) (err error) {

	var imagef string

	if i.video {
		// the webm thumbnail source is the mjpeg frame ffmpeg already extracted
		imagef = fmt.Sprintf("jpg:%s[0]", i.Thumbpath)
	} else {
		// pin the coder to the validated type so convert cannot be coerced into
		// a different coder by crafted content
		imagef = fmt.Sprintf("%s:%s[0]", magickCoder(i.Ext), i.Filepath)
	}

	originalDimensions := fmt.Sprintf("%dx%d", i.OrigWidth, i.OrigHeight)

	// start every invocation with resource limits
	args := append([]string{}, magickLimits...)

	// different options for avatars
	if i.avatar {
		args = append(args,
			"-size",
			originalDimensions,
			imagef,
			"-background",
			"none",
			"-thumbnail",
			fmt.Sprintf("%dx%d^", maxwidth, maxheight),
			"-gravity",
			"center",
			"-extent",
			fmt.Sprintf("%dx%d", maxwidth, maxheight),
			i.Thumbpath,
		)
	} else {
		args = append(args,
			"-background",
			"white",
			"-flatten",
			"-size",
			originalDimensions,
			"-resize",
			fmt.Sprintf("%dx%d>", maxwidth, maxheight),
			"-quality",
			"90",
			imagef,
			i.Thumbpath,
		)
	}

	// Create context with timeout for ImageMagick operations
	ctx, cancel := context.WithTimeout(context.Background(), processTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "convert", args...)
	_, err = cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("thumbnail creation timed out after %v", processTimeout)
		}
		return errors.New("problem making thumbnail")
	}

	thumbpath := local.Settings.Directories.ThumbnailDir

	if i.avatar {
		thumbpath = local.Settings.Directories.AvatarDir
	}

	thumb, err := os.OpenInRoot(thumbpath, i.Thumbnail)
	if err != nil {
		return errors.New("problem creating thumbnail file")
	}
	defer thumb.Close()

	img, _, err := image.DecodeConfig(thumb)
	if err != nil {
		return errors.New("problem decoding thumbnail")
	}

	i.ThumbWidth = img.Width
	i.ThumbHeight = img.Height

	return
}

// removeOriginal deletes the full-size source file that saveFile wrote to the
// image directory. The avatar pipeline serves only the generated
// AvatarDir/<id>.png thumbnail, so the ImageDir original would otherwise be
// orphaned (and accumulate unbounded) on every avatar upload/generation. Must
// be called AFTER createThumbnail, since that step reads its source from
// i.Filepath (the ImageDir original).
func (i *ImageType) removeOriginal() {
	if i.Filename == "" {
		return
	}

	// Open the image dir with traversal-resistant root
	root, err := os.OpenRoot(local.Settings.Directories.ImageDir)
	if err != nil {
		return
	}
	defer root.Close()

	// Only remove an actual file at the expected name
	if info, err := root.Stat(i.Filename); err == nil && !info.IsDir() {
		_ = root.Remove(i.Filename)
	}
}

// cleanupFiles removes any files that were created during the image processing
// to ensure we don't leave orphaned files on disk after failed operations
func (i *ImageType) cleanupFiles() {
	// Clean up the main image file if it exists and has a valid path
	if i.Filepath != "" && i.Filename != "" {
		// Determine which root directory to use
		rootDir := local.Settings.Directories.ImageDir
		if i.avatar {
			rootDir = local.Settings.Directories.AvatarDir
		}

		// Open the root directory
		root, err := os.OpenRoot(rootDir)
		if err == nil {
			defer root.Close()

			// Extra validation to ensure we're only dealing with files
			fileInfo, err := root.Stat(i.Filename)
			if err == nil && !fileInfo.IsDir() {
				// Validate that the path ends with the filename
				if strings.HasSuffix(i.Filepath, i.Filename) {
					// Safe to remove using the root-based approach
					_ = root.Remove(i.Filename)
				}
			}
		}
	}

	// Clean up the thumbnail if it exists and has a valid path
	if i.Thumbpath != "" && i.Thumbnail != "" {
		// Determine which root directory to use
		thumbRootDir := local.Settings.Directories.ThumbnailDir
		if i.avatar {
			thumbRootDir = local.Settings.Directories.AvatarDir
		}

		// Open the root directory
		thumbRoot, err := os.OpenRoot(thumbRootDir)
		if err == nil {
			defer thumbRoot.Close()

			// Extra validation to ensure we're only dealing with files
			fileInfo, err := thumbRoot.Stat(i.Thumbnail)
			if err == nil && !fileInfo.IsDir() {
				// Validate that the path ends with the thumbnail name
				if strings.HasSuffix(i.Thumbpath, i.Thumbnail) {
					// Safe to remove using the root-based approach
					_ = thumbRoot.Remove(i.Thumbnail)
				}
			}
		}
	}
}
