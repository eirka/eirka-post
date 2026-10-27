package utils

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/eirka/eirka-libs/config"

	"github.com/eirka/eirka-post/akismet"
)

func TestAkismetError(t *testing.T) {
	// the shape of a client.Do failure with the old key-in-hostname endpoint
	keyURLErr := &url.Error{
		Op:  "Post",
		URL: "http://0123456789ab.rest.akismet.com/1.1/comment-check",
		Err: errors.New("dial tcp 192.0.2.1:80: i/o timeout"),
	}

	tests := []struct {
		name    string
		err     error
		want    error
		wantLog string
	}{
		{name: "no error", err: nil, want: nil},
		{name: "spam", err: akismet.ErrSpam, want: akismet.ErrSpam},
		{name: "wrapped spam", err: fmt.Errorf("check: %w", akismet.ErrSpam), want: akismet.ErrSpam},
		{name: "transport error", err: keyURLErr, want: ErrAkismetFailed, wantLog: "i/o timeout"},
		{name: "invalid key", err: errors.New("key invalid"), want: ErrAkismetFailed, wantLog: "key invalid"},
		{name: "unknown response", err: errors.New("unknown response"), want: ErrAkismetFailed, wantLog: "unknown response"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			log.SetOutput(&buf)
			defer log.SetOutput(os.Stderr)

			got := akismetError(tt.err)

			assert.Equal(t, tt.want, got)
			if got != nil {
				assert.NotContains(t, got.Error(), "0123456789ab", "client-facing error must not carry the api key")
			}

			if tt.wantLog != "" {
				assert.Contains(t, buf.String(), tt.wantLog, "the underlying error should be logged server-side")
			} else {
				assert.Empty(t, buf.String(), "nothing should be logged")
			}
		})
	}
}

func TestAkismetCheckNotConfigured(t *testing.T) {
	orig := config.Settings.Akismet.Configured
	config.Settings.Akismet.Configured = false
	defer func() { config.Settings.Akismet.Configured = orig }()

	a := Akismet{IP: "192.0.2.1", Comment: "hello"}
	assert.NoError(t, a.Check(), "check is a noop when akismet is not configured")
}

// roundTripFunc stands in for the transport used by the akismet client
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// TestAkismetCheck runs Check end to end with akismet configured. The akismet
// package builds its http.Client without a Transport, so replacing
// http.DefaultTransport keeps every request in process.
func TestAkismetCheck(t *testing.T) {
	const key = "0123456789ab"

	origSettings := config.Settings.Akismet
	origTransport := http.DefaultTransport
	defer func() {
		config.Settings.Akismet = origSettings
		http.DefaultTransport = origTransport
	}()

	config.Settings.Akismet.Configured = true
	config.Settings.Akismet.Key = key
	config.Settings.Akismet.Host = "https://example.com"

	// an empty body stands for a transport failure on that endpoint
	tests := []struct {
		name   string
		verify string
		check  string
		want   error
	}{
		{name: "not spam", verify: "valid", check: "false", want: nil},
		{name: "spam", verify: "valid", check: "true", want: akismet.ErrSpam},
		{name: "verify-key transport error", verify: "", check: "false", want: ErrAkismetFailed},
		{name: "invalid key", verify: "invalid", check: "false", want: ErrAkismetFailed},
		{name: "comment-check transport error", verify: "valid", check: "", want: ErrAkismetFailed},
		{name: "comment-check invalid", verify: "valid", check: "invalid", want: ErrAkismetFailed},
		{name: "comment-check unknown response", verify: "valid", check: "something else", want: ErrAkismetFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				body := tt.verify
				if strings.HasSuffix(r.URL.Path, "/comment-check") {
					body = tt.check
				}
				if body == "" {
					return nil, errors.New("dial tcp 192.0.2.1:443: i/o timeout")
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(body)),
					Request:    r,
				}, nil
			})

			var buf bytes.Buffer
			log.SetOutput(&buf)
			defer log.SetOutput(os.Stderr)

			a := Akismet{IP: "192.0.2.1", Ua: "test-agent", Comment: "hello"}
			got := a.Check()

			assert.Equal(t, tt.want, got)
			if got != nil {
				assert.NotContains(t, got.Error(), key, "client-facing error must not carry the api key")
			}
			if tt.want == ErrAkismetFailed {
				assert.Contains(t, buf.String(), "akismet: ", "the underlying error should be logged server-side")
			}
			assert.NotContains(t, buf.String(), key, "the server log must not carry the api key")
		})
	}
}
