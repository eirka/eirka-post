.PHONY: build lint preflight test check

build:
	go build ./...

lint:
	go vet ./...

preflight:
	@command -v ffmpeg >/dev/null 2>&1 || { echo "preflight FAILED: ffmpeg not on PATH (utils init() panics without it)" >&2; exit 1; }
	@command -v ffprobe >/dev/null 2>&1 || { echo "preflight FAILED: ffprobe not on PATH (utils init() panics without it)" >&2; exit 1; }
	@command -v convert >/dev/null 2>&1 || { echo "preflight FAILED: convert (ImageMagick) not on PATH (utils init() panics without it)" >&2; exit 1; }

test: preflight
	go test -count=1 ./...

check: build lint test
