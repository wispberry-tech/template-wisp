package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// hashedFilename matches ".{8-hex-chars}.{ext}" at the end of a filename —
// the output of Builder hashing. Used to decide cache headers.
var hashedFilename = regexp.MustCompile(`\.[0-9a-f]{8}\.[^.]+$`)

// Handler returns an http.Handler serving files from OutputDir with path-safe
// lookup and smart cache headers. Hashed files get long immutable caching;
// non-hashed files get ETag + must-revalidate.
//
// The handler expects URLPrefix to already be stripped from the incoming
// request path (use Route() for one-line wiring).
func (b *Builder) Handler() http.Handler {
	outAbs, err := filepath.Abs(b.cfg.OutputDir)
	if err != nil {
		// Fall back to the raw path; OS will return errors consistently.
		outAbs = b.cfg.OutputDir
	}
	// Best-effort real-path resolution so symlink equality comparisons work.
	if realOut, err := filepath.EvalSymlinks(outAbs); err == nil {
		outAbs = realOut
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqPath := r.URL.Path

		// Reject null bytes outright.
		if strings.ContainsRune(reqPath, 0) {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		// Reject any traversal segment.
		if strings.Contains(reqPath, "..") {
			http.NotFound(w, r)
			return
		}

		// Clean, join, and verify the result stays inside outAbs.
		clean := filepath.Clean("/" + reqPath)
		full := filepath.Join(outAbs, filepath.FromSlash(clean))
		if !strings.HasPrefix(full, outAbs+string(os.PathSeparator)) && full != outAbs {
			http.NotFound(w, r)
			return
		}

		// Resolve symlinks and re-check prefix.
		realFull, err := filepath.EvalSymlinks(full)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if !strings.HasPrefix(realFull, outAbs+string(os.PathSeparator)) && realFull != outAbs {
			http.NotFound(w, r)
			return
		}

		info, err := os.Stat(realFull)
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}

		// Headers.
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if ct := mime.TypeByExtension(filepath.Ext(realFull)); ct != "" {
			w.Header().Set("Content-Type", ct)
		}

		name := filepath.Base(realFull)
		if hashedFilename.MatchString(name) {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			etag := computeETag(info.ModTime().UnixNano(), info.Size())
			w.Header().Set("ETag", etag)
			w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
			if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
		w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))

		f, err := os.Open(realFull)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()

		if r.Method == http.MethodHead {
			return
		}
		_, _ = io.Copy(w, f)
	})
}

// Route returns the (pattern, handler) pair for mounting under URLPrefix.
// Equivalent to ("{URLPrefix}/", http.StripPrefix("{URLPrefix}/", b.Handler())).
//
// Example:
//
//	mux.Handle(builder.Route())
func (b *Builder) Route() (string, http.Handler) {
	prefix := strings.TrimSuffix(b.cfg.URLPrefix, "/") + "/"
	return prefix, http.StripPrefix(prefix, b.Handler())
}

func computeETag(mtimeNanos, size int64) string {
	h := sha256.New()
	fmt.Fprintf(h, "%d-%d", mtimeNanos, size)
	return `"` + hex.EncodeToString(h.Sum(nil)[:12]) + `"`
}
