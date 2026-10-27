package akismet

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testKey = "0123456789ab"

func testConfig() *Config {
	return &Config{
		APIKey:    testKey,
		Host:      "https://example.com",
		UserAgent: UserAgentString("Pram/test"),
	}
}

// useServer points the package at a local server for the duration of the test
func useServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	orig := apiBase
	apiBase = srv.URL + "/1.1"
	t.Cleanup(func() { apiBase = orig })
}

func TestEndpointURLs(t *testing.T) {
	config := testConfig()

	urls := map[string]string{
		"verify-key":    config.VerifyKeyURL(),
		"comment-check": config.CommentCheckURL(),
		"submit-spam":   config.SubmitSpamURL(),
		"submit-ham":    config.SubmitHamURL(),
	}

	for name, u := range urls {
		assert.Equal(t, "https://rest.akismet.com/1.1/"+name, u, "endpoint must be https on the shared host")
		assert.NotContains(t, u, testKey, "api key must not appear in the URL")
	}
}

func TestCommentCheck(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     error
	}{
		{name: "ham", response: "false", want: nil},
		{name: "spam", response: "true", want: ErrSpam},
		{name: "invalid request", response: "invalid", want: errInvalidRequest},
		{name: "unexpected body", response: "something else", want: errUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			var gotForm url.Values

			useServer(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				assert.NoError(t, r.ParseForm())
				gotForm = r.PostForm
				w.Write([]byte(tt.response))
			})

			err := CommentCheck(testConfig(), Comment{
				UserIP:    "192.0.2.1",
				UserAgent: "test-agent",
				Type:      "comment",
				Content:   "hello",
			})

			assert.Equal(t, tt.want, err)
			assert.Equal(t, "/1.1/comment-check", gotPath)
			assert.Equal(t, testKey, gotForm.Get("api_key"), "api key must be sent in the body")
			assert.Equal(t, "192.0.2.1", gotForm.Get("user_ip"))
			assert.Equal(t, "hello", gotForm.Get("comment_content"))
		})
	}
}

func TestVerifyKey(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     error
	}{
		{name: "valid", response: "valid", want: nil},
		{name: "invalid", response: "invalid", want: errInvalidKey},
		{name: "unexpected body", response: "something else", want: errUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			var gotForm url.Values

			useServer(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				assert.NoError(t, r.ParseForm())
				gotForm = r.PostForm
				w.Write([]byte(tt.response))
			})

			err := VerifyKey(testConfig())

			assert.Equal(t, tt.want, err)
			assert.Equal(t, "/1.1/verify-key", gotPath)
			assert.Equal(t, testKey, gotForm.Get("api_key"), "api key must be sent in the body")
			assert.Equal(t, "https://example.com", gotForm.Get("blog"))
		})
	}
}

// A transport failure returns a *url.Error, whose message includes the request
// URL; the API key must not be in it.
func TestTransportErrorOmitsKey(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close() // connections are now refused

	orig := apiBase
	apiBase = srv.URL + "/1.1"
	t.Cleanup(func() { apiBase = orig })

	calls := map[string]func() error{
		"VerifyKey":    func() error { return VerifyKey(testConfig()) },
		"CommentCheck": func() error { return CommentCheck(testConfig(), Comment{UserIP: "192.0.2.1"}) },
	}

	for name, call := range calls {
		err := call()
		require.Error(t, err, name)

		var urlErr *url.Error
		assert.ErrorAs(t, err, &urlErr, "%s should fail in the transport", name)
		assert.False(t, strings.Contains(err.Error(), testKey), "%s error leaks the api key: %s", name, err)
	}
}

// User-controlled fields must reach akismet exactly as sent, without injecting
// or overriding other parameters.
func TestCommentCheckEncodesBody(t *testing.T) {
	tests := []struct {
		name    string
		content string
		ua      string
		ref     string
	}{
		{name: "parameter injection", content: "hi&user_ip=203.0.113.9&is_test=1", ua: "agent&comment_type=pingback", ref: "https://ref.example/?a=1&blog=https://evil.example"},
		{name: "equals sign", content: "a=b", ua: "k=v", ref: "https://ref.example/?q=a=b"},
		{name: "plus and percent", content: "1+1 is 100% %41", ua: "a+b", ref: "https://ref.example/%20"},
		{name: "newline and unicode", content: "line one\nline two ✓ 日本語", ua: "agent\r\nX: y", ref: "https://ref.example/ü"},
	}

	// every parameter MakePOST sends, each exactly once
	wantKeys := []string{
		"api_key", "blog", "user_ip", "user_agent", "referrer", "permalink",
		"comment_type", "comment_author", "comment_author_email", "comment_author_url", "comment_content",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotForm url.Values

			useServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.NoError(t, r.ParseForm())
				gotForm = r.PostForm
				w.Write([]byte("false"))
			})

			err := CommentCheck(testConfig(), Comment{
				UserIP:    "192.0.2.1",
				UserAgent: tt.ua,
				Referrer:  tt.ref,
				Type:      "comment",
				Content:   tt.content,
			})
			assert.NoError(t, err)

			gotKeys := make([]string, 0, len(gotForm))
			for k, v := range gotForm {
				gotKeys = append(gotKeys, k)
				assert.Len(t, v, 1, "parameter %q must be sent once", k)
			}
			assert.ElementsMatch(t, wantKeys, gotKeys, "no parameters may be injected")

			assert.Equal(t, tt.content, gotForm.Get("comment_content"))
			assert.Equal(t, tt.ua, gotForm.Get("user_agent"))
			assert.Equal(t, tt.ref, gotForm.Get("referrer"))
			assert.Equal(t, "192.0.2.1", gotForm.Get("user_ip"))
			assert.Equal(t, "comment", gotForm.Get("comment_type"))
			assert.Equal(t, "https://example.com", gotForm.Get("blog"))
			assert.Equal(t, testKey, gotForm.Get("api_key"))
		})
	}
}
