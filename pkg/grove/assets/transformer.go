// Package assets is Grove's opt-in asset build pipeline. It scans a source
// directory for CSS and JS files, applies a pluggable Transformer, writes
// content-hashed copies to an output directory, and produces a JSON manifest
// mapping logical asset names to served URLs.
//
// The engine resolves {% asset %} logical names through a resolver function
// (typically Manifest.Resolve) configured via grove.WithAssetResolver. Apps
// that do not import this package pay zero cost.
package assets

// Transformer processes raw asset bytes for a given media type.
//
// mediaType is "text/css" or "application/javascript". Implementations should
// return src unchanged for unknown types, or return an error if strict.
type Transformer interface {
	Transform(src []byte, mediaType string) ([]byte, error)
}

// NoopTransformer returns input unchanged. It is the default when no
// transformer is configured.
type NoopTransformer struct{}

// Transform returns src unchanged.
func (NoopTransformer) Transform(src []byte, _ string) ([]byte, error) {
	return src, nil
}
