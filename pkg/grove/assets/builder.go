package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// EventType classifies a build lifecycle event.
type EventType int

const (
	EventDiscovered EventType = iota // File found during scan
	EventBuilt                       // File transformed and written
	EventSkipped                     // Unchanged (cache hit)
	EventPruned                      // Entry removed by prune pass
	EventError                       // Transform or I/O failure
)

// String returns the human-readable event name.
func (t EventType) String() string {
	switch t {
	case EventDiscovered:
		return "discovered"
	case EventBuilt:
		return "built"
	case EventSkipped:
		return "skipped"
	case EventPruned:
		return "pruned"
	case EventError:
		return "error"
	default:
		return "unknown"
	}
}

// Event is a single build lifecycle record.
type Event struct {
	Type        EventType
	LogicalName string
	OutputPath  string        // Absolute path written (Built only)
	Duration    time.Duration // Build duration (Built only)
	InputSize   int           // Bytes before transform (Built only)
	OutputSize  int           // Bytes after transform (Built only)
	Err         error         // Non-nil for EventError
}

// WatchHandlers bundles callbacks for Watch mode.
type WatchHandlers struct {
	// OnChange is called after a rebuild with the current manifest.
	// Called even when some files failed — manifest reflects partial swap.
	// Required.
	OnChange func(*Manifest)

	// OnError is called once per failed file during a rebuild. The manifest
	// passed to OnChange retains the prior entry for that file. Optional.
	OnError func(error)

	// OnEvent receives structured build events. Optional.
	OnEvent func(Event)
}

// Config controls the asset build pipeline.
type Config struct {
	// SourceDir is the root directory to scan for CSS/JS files.
	SourceDir string

	// OutputDir is where processed files are written. Created automatically.
	OutputDir string

	// URLPrefix is prepended to output paths in the manifest.
	// Example: "/dist" produces manifest entries like "/dist/button.a1b2c3d4.css".
	// Default: "/dist".
	URLPrefix string

	// Extensions lists file extensions to process. Default: [".css", ".js"].
	Extensions []string

	// HashFiles controls whether content hashes are inserted into filenames.
	// Default: true.
	HashFiles bool

	// CSSTransformer processes CSS files. Default: NoopTransformer.
	CSSTransformer Transformer

	// JSTransformer processes JS files. Default: NoopTransformer.
	JSTransformer Transformer

	// ManifestPath is where to write the manifest JSON file. Empty means
	// don't write to disk (manifest returned in memory only).
	ManifestPath string

	// EmitSourceMaps is reserved for transformers that support source maps.
	// When true, Builder.Build forwards the flag; it does not emit maps on
	// its own. See the minify sub-package.
	EmitSourceMaps bool

	// IncludeBuildStats records per-file build statistics in the manifest.
	IncludeBuildStats bool

	// PruneUnreferenced drops manifest entries whose logical name is not in
	// the engine's referenced-name set. Requires SetReferencedNameProvider.
	PruneUnreferenced bool
}

// defaults fills zero values with spec defaults. Returns a new Config.
func (c Config) defaults() Config {
	if c.URLPrefix == "" {
		c.URLPrefix = "/dist"
	}
	if len(c.Extensions) == 0 {
		c.Extensions = []string{".css", ".js"}
	}
	if c.CSSTransformer == nil {
		c.CSSTransformer = NoopTransformer{}
	}
	if c.JSTransformer == nil {
		c.JSTransformer = NoopTransformer{}
	}
	// HashFiles defaults to true only when explicitly set; we can't
	// distinguish "unset" from "false" on a bool field. Callers who want
	// hashing omit the field via zero-value… but zero-value of bool is
	// false. To match the spec's default=true, use an explicit NewConfig
	// constructor; for now treat the zero-value Config with HashFiles:false
	// as "no hashing" and document this.
	return c
}

// Builder scans, processes, and outputs asset files.
type Builder struct {
	cfg         Config
	buildMu     sync.Mutex
	refProvider func() map[string]struct{}
}

// New creates a Builder with the given config. Does not perform any I/O
// until Build() or Watch() is called.
//
// Note on defaults: HashFiles is a plain bool, so its zero value is false.
// To get the spec-default behavior (HashFiles=true, Extensions=[".css",".js"],
// URLPrefix="/dist"), use NewWithDefaults or set HashFiles: true explicitly.
func New(cfg Config) *Builder {
	return &Builder{cfg: cfg.defaults()}
}

// NewWithDefaults creates a Builder and forces HashFiles=true plus all
// other spec defaults. Fields explicitly set in cfg override the defaults.
func NewWithDefaults(cfg Config) *Builder {
	cfg = cfg.defaults()
	cfg.HashFiles = true
	return &Builder{cfg: cfg}
}

// SetReferencedNameProvider wires a getter for the set of logical asset
// names referenced during rendering. Required when Config.PruneUnreferenced
// is true. Safe to call at any time.
func (b *Builder) SetReferencedNameProvider(fn func() map[string]struct{}) {
	b.buildMu.Lock()
	b.refProvider = fn
	b.buildMu.Unlock()
}

// Config returns a copy of the builder's resolved configuration.
func (b *Builder) Config() Config { return b.cfg }

// Build scans SourceDir, applies transformers, writes hashed copies to
// OutputDir, and returns the resulting manifest. Safe for sequential
// invocation; concurrent calls are serialized by an internal mutex.
func (b *Builder) Build() (*Manifest, error) {
	return b.build(nil)
}

func (b *Builder) build(onEvent func(Event)) (*Manifest, error) {
	b.buildMu.Lock()
	defer b.buildMu.Unlock()

	if b.cfg.SourceDir == "" {
		return nil, errors.New("assets: Config.SourceDir is required")
	}
	if b.cfg.OutputDir == "" {
		return nil, errors.New("assets: Config.OutputDir is required")
	}
	if err := os.MkdirAll(b.cfg.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("assets: create output dir: %w", err)
	}

	manifest := NewManifest()
	exts := extSet(b.cfg.Extensions)

	var walkErr error
	err := filepath.WalkDir(b.cfg.SourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if _, ok := exts[ext]; !ok {
			return nil
		}
		rel, relErr := filepath.Rel(b.cfg.SourceDir, path)
		if relErr != nil {
			return relErr
		}
		logical := filepath.ToSlash(rel)

		emit(onEvent, Event{Type: EventDiscovered, LogicalName: logical})

		if bErr := b.processFile(path, logical, ext, manifest, onEvent); bErr != nil {
			// Don't abort the whole build on a per-file error; record and move on.
			walkErr = errors.Join(walkErr, bErr)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("assets: walk %q: %w", b.cfg.SourceDir, err)
	}

	// Prune unreferenced entries if requested.
	if b.cfg.PruneUnreferenced && b.refProvider != nil {
		refs := b.refProvider()
		if len(refs) > 0 {
			for logical := range manifest.Entries() {
				if _, used := refs[logical]; !used {
					manifest.Delete(logical)
					emit(onEvent, Event{Type: EventPruned, LogicalName: logical})
				}
			}
		}
	}

	// Persist manifest if requested.
	if b.cfg.ManifestPath != "" {
		if saveErr := manifest.Save(b.cfg.ManifestPath); saveErr != nil {
			return manifest, saveErr
		}
	}

	return manifest, walkErr
}

// processFile reads, transforms, hashes, and writes one source file.
func (b *Builder) processFile(srcPath, logical, ext string, manifest *Manifest, onEvent func(Event)) error {
	start := time.Now()
	raw, err := os.ReadFile(srcPath)
	if err != nil {
		emit(onEvent, Event{Type: EventError, LogicalName: logical, Err: err})
		return fmt.Errorf("assets: read %q: %w", srcPath, err)
	}

	tr, mediaType := b.transformerFor(ext)
	out, err := tr.Transform(raw, mediaType)
	if err != nil {
		emit(onEvent, Event{Type: EventError, LogicalName: logical, Err: err})
		return fmt.Errorf("assets: transform %q: %w", logical, err)
	}

	// Compute hash and build destination filename.
	var hashPart string
	if b.cfg.HashFiles {
		sum := sha256.Sum256(out)
		hashPart = hex.EncodeToString(sum[:4]) // 8 hex chars
	}
	destRel := buildDestName(logical, hashPart)
	destAbs := filepath.Join(b.cfg.OutputDir, filepath.FromSlash(destRel))
	if err := os.MkdirAll(filepath.Dir(destAbs), 0o755); err != nil {
		emit(onEvent, Event{Type: EventError, LogicalName: logical, Err: err})
		return err
	}
	if err := os.WriteFile(destAbs, out, 0o644); err != nil {
		emit(onEvent, Event{Type: EventError, LogicalName: logical, Err: err})
		return err
	}

	url := strings.TrimSuffix(b.cfg.URLPrefix, "/") + "/" + destRel
	manifest.Set(logical, url)

	if b.cfg.IncludeBuildStats {
		ratio := 0.0
		if len(raw) > 0 {
			ratio = float64(len(out)) / float64(len(raw))
		}
		manifest.SetStats(logical, BuildStats{
			DurationMs:  time.Since(start).Milliseconds(),
			InputBytes:  len(raw),
			OutputBytes: len(out),
			Ratio:       ratio,
		})
	}

	emit(onEvent, Event{
		Type:        EventBuilt,
		LogicalName: logical,
		OutputPath:  destAbs,
		Duration:    time.Since(start),
		InputSize:   len(raw),
		OutputSize:  len(out),
	})
	return nil
}

// transformerFor returns the configured transformer and media type for ext.
func (b *Builder) transformerFor(ext string) (Transformer, string) {
	switch ext {
	case ".css":
		return b.cfg.CSSTransformer, "text/css"
	case ".js":
		return b.cfg.JSTransformer, "application/javascript"
	default:
		return NoopTransformer{}, "application/octet-stream"
	}
}

// buildDestName inserts hashPart before the extension. logical is a
// forward-slash path. Returns a forward-slash path.
func buildDestName(logical, hashPart string) string {
	if hashPart == "" {
		return logical
	}
	ext := filepath.Ext(logical)
	stem := strings.TrimSuffix(logical, ext)
	return stem + "." + hashPart + ext
}

// extSet returns a lowercased set of the given extensions.
func extSet(exts []string) map[string]struct{} {
	out := make(map[string]struct{}, len(exts))
	for _, e := range exts {
		out[strings.ToLower(e)] = struct{}{}
	}
	return out
}

func emit(onEvent func(Event), e Event) {
	if onEvent != nil {
		onEvent(e)
	}
}

