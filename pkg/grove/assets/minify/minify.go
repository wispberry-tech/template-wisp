// Package minify provides a MinifyTransformer that implements
// assets.Transformer using github.com/tdewolff/minify/v2 for CSS and JS.
//
// Apps opt in by importing this package and passing the transformer to
// assets.Config. The core pkg/grove/assets has no dependency on this
// sub-package, so apps that don't want the tdewolff/minify dependency can
// use NoopTransformer or implement their own.
package minify

import (
	"bytes"
	"fmt"

	"github.com/tdewolff/minify/v2"
	mcss "github.com/tdewolff/minify/v2/css"
	mjs "github.com/tdewolff/minify/v2/js"
)

// MinifyTransformer minifies CSS and JS using tdewolff/minify. It
// dispatches on the mediaType argument passed to Transform.
type MinifyTransformer struct {
	m *minify.M
}

// New constructs a MinifyTransformer with CSS and JS minifiers registered.
func New() *MinifyTransformer {
	m := minify.New()
	m.AddFunc("text/css", mcss.Minify)
	m.AddFunc("application/javascript", mjs.Minify)
	return &MinifyTransformer{m: m}
}

// Transform minifies src for the given mediaType. Returns an error for
// unsupported media types so callers don't silently ship unminified output.
func (t *MinifyTransformer) Transform(src []byte, mediaType string) ([]byte, error) {
	switch mediaType {
	case "text/css", "application/javascript":
		var buf bytes.Buffer
		if err := t.m.Minify(mediaType, &buf, bytes.NewReader(src)); err != nil {
			return nil, fmt.Errorf("minify: %s: %w", mediaType, err)
		}
		return buf.Bytes(), nil
	default:
		return nil, fmt.Errorf("minify: unsupported media type %q", mediaType)
	}
}
