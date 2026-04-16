package assets

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func setupServeBuilder(t *testing.T) (*Builder, string) {
	t.Helper()
	src := t.TempDir()
	dst := t.TempDir()
	writeTree(t, src, map[string]string{
		"app.css":        ".app{}",
		"nested/site.js": "console.log(1);",
	})
	b := NewWithDefaults(Config{SourceDir: src, OutputDir: dst})
	_, err := b.Build()
	require.NoError(t, err)
	return b, dst
}

func TestHandler_ServesHashedFileWithImmutableCache(t *testing.T) {
	b, _ := setupServeBuilder(t)
	m, err := b.Build()
	require.NoError(t, err)
	url := m.Entries()["app.css"]
	pathOnly := url[len("/dist"):]

	srv := httptest.NewServer(http.StripPrefix("/dist", b.Handler()))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/dist" + pathOnly)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Cache-Control"), "immutable")
	require.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))

	body, _ := io.ReadAll(resp.Body)
	require.Equal(t, ".app{}", string(body))
}

func TestHandler_NonHashedGetsETag(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeTree(t, src, map[string]string{"plain.css": ".p{}"})
	b := New(Config{SourceDir: src, OutputDir: dst, HashFiles: false})
	_, err := b.Build()
	require.NoError(t, err)

	srv := httptest.NewServer(http.StripPrefix("/dist", b.Handler()))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/dist/plain.css")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	etag := resp.Header.Get("ETag")
	require.NotEmpty(t, etag)
	require.Contains(t, resp.Header.Get("Cache-Control"), "must-revalidate")

	// Conditional GET returns 304.
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/dist/plain.css", nil)
	require.NoError(t, err)
	req.Header.Set("If-None-Match", etag)
	resp2, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp2.Body.Close()
	require.Equal(t, http.StatusNotModified, resp2.StatusCode)
}

func TestHandler_RejectsTraversal(t *testing.T) {
	b, dst := setupServeBuilder(t)
	// Place a sensitive file OUTSIDE the output dir.
	secret := filepath.Join(filepath.Dir(dst), "secret.txt")
	require.NoError(t, os.WriteFile(secret, []byte("top-secret"), 0o644))

	srv := httptest.NewServer(http.StripPrefix("/dist", b.Handler()))
	defer srv.Close()

	// Various traversal attempts.
	cases := []string{
		"/dist/../secret.txt",
		"/dist/..%2fsecret.txt",
		"/dist/foo/../../secret.txt",
	}
	for _, c := range cases {
		resp, err := http.Get(srv.URL + c)
		require.NoError(t, err, c)
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		require.NotContains(t, string(body), "top-secret", "traversal leaked on %s", c)
		require.NotEqual(t, http.StatusOK, resp.StatusCode, "traversal succeeded on %s", c)
	}
}

func TestHandler_RejectsNullByte(t *testing.T) {
	b, _ := setupServeBuilder(t)
	srv := httptest.NewServer(http.StripPrefix("/dist", b.Handler()))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/dist/app.css", nil)
	require.NoError(t, err)
	// URL.Path intentionally contains a null byte.
	req.URL.Path = "/dist/app\x00.css"
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.NotEqual(t, http.StatusOK, resp.StatusCode)
}

func TestHandler_404OnMissing(t *testing.T) {
	b, _ := setupServeBuilder(t)
	srv := httptest.NewServer(http.StripPrefix("/dist", b.Handler()))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/dist/nope.css")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestHandler_HEADOmitsBody(t *testing.T) {
	b, _ := setupServeBuilder(t)
	m, _ := b.Build()
	pathOnly := m.Entries()["app.css"][len("/dist"):]

	srv := httptest.NewServer(http.StripPrefix("/dist", b.Handler()))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodHead, srv.URL+"/dist"+pathOnly, nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	require.Empty(t, body)
}

func TestBuilder_Route(t *testing.T) {
	b, _ := setupServeBuilder(t)
	m, err := b.Build()
	require.NoError(t, err)

	pattern, h := b.Route()
	require.Equal(t, "/dist/", pattern)

	mux := http.NewServeMux()
	mux.Handle(pattern, h)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	url := srv.URL + m.Entries()["app.css"]
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestBuilder_CustomURLPrefixRoute(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeTree(t, src, map[string]string{"a.css": ".a{}"})

	b := NewWithDefaults(Config{SourceDir: src, OutputDir: dst, URLPrefix: "/static/built"})
	m, err := b.Build()
	require.NoError(t, err)

	pattern, h := b.Route()
	require.Equal(t, "/static/built/", pattern)

	mux := http.NewServeMux()
	mux.Handle(pattern, h)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + m.Entries()["a.css"])
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
