package assets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// BuildStats records per-file build telemetry. Populated when
// Config.IncludeBuildStats is true.
type BuildStats struct {
	DurationMs  int64   `json:"duration_ms"`
	InputBytes  int     `json:"input_bytes"`
	OutputBytes int     `json:"output_bytes"`
	Ratio       float64 `json:"ratio"`
}

// Manifest maps logical asset names to served URLs, plus optional sibling
// data (source map URLs, build stats). Safe for concurrent reads; Resolve
// and Entries may be called from multiple goroutines.
type Manifest struct {
	mu      sync.RWMutex
	entries map[string]string
	sources map[string]string
	stats   map[string]BuildStats
}

// NewManifest returns an empty Manifest.
func NewManifest() *Manifest {
	return &Manifest{
		entries: make(map[string]string),
		sources: make(map[string]string),
		stats:   make(map[string]BuildStats),
	}
}

// Resolve returns the served URL for a logical name. When not found, callers
// should fall back to the original name. The signature matches grove.AssetResolver,
// so manifest.Resolve can be passed directly to grove.WithAssetResolver:
//
//	eng := grove.New(grove.WithAssetResolver(manifest.Resolve))
func (m *Manifest) Resolve(logicalName string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	url, ok := m.entries[logicalName]
	return url, ok
}

// Entries returns a copy of the canonical logical-to-URL map.
func (m *Manifest) Entries() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]string, len(m.entries))
	for k, v := range m.entries {
		out[k] = v
	}
	return out
}

// Sources returns a copy of the logical-name-to-sourcemap-URL map.
func (m *Manifest) Sources() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]string, len(m.sources))
	for k, v := range m.sources {
		out[k] = v
	}
	return out
}

// Stats returns a copy of the per-file build-stats map.
func (m *Manifest) Stats() map[string]BuildStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]BuildStats, len(m.stats))
	for k, v := range m.stats {
		out[k] = v
	}
	return out
}

// Set records a single logical-to-URL mapping. Used internally by the
// Builder and exposed for callers that want to hand-assemble a manifest
// (e.g. CDN-only setups).
func (m *Manifest) Set(logicalName, url string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.entries == nil {
		m.entries = make(map[string]string)
	}
	m.entries[logicalName] = url
}

// SetSource records a source-map URL for a logical asset.
func (m *Manifest) SetSource(logicalName, mapURL string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sources == nil {
		m.sources = make(map[string]string)
	}
	m.sources[logicalName] = mapURL
}

// SetStats records build statistics for a logical asset.
func (m *Manifest) SetStats(logicalName string, s BuildStats) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stats == nil {
		m.stats = make(map[string]BuildStats)
	}
	m.stats[logicalName] = s
}

// Delete removes all data (entry, source, stats) for a logical name.
func (m *Manifest) Delete(logicalName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, logicalName)
	delete(m.sources, logicalName)
	delete(m.stats, logicalName)
}

// manifestFile is the on-disk JSON schema. The legacy bare-map form is
// detected by probing for the "assets" key before unmarshaling.
type manifestFile struct {
	Assets  map[string]string     `json:"assets,omitempty"`
	Sources map[string]string     `json:"sources,omitempty"`
	Stats   map[string]BuildStats `json:"stats,omitempty"`
}

// LoadManifest reads a manifest from a JSON file. It auto-detects the
// structured format (with "assets" top-level key) and the legacy bare-map
// form ({logical: url}).
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("assets: read manifest %q: %w", path, err)
	}
	return parseManifest(data)
}

func parseManifest(data []byte) (*Manifest, error) {
	// Probe for "assets" key to decide between structured and legacy form.
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("assets: parse manifest: %w", err)
	}
	m := NewManifest()
	if _, hasAssets := probe["assets"]; hasAssets {
		var f manifestFile
		if err := json.Unmarshal(data, &f); err != nil {
			return nil, fmt.Errorf("assets: parse manifest: %w", err)
		}
		if f.Assets != nil {
			m.entries = f.Assets
		}
		if f.Sources != nil {
			m.sources = f.Sources
		}
		if f.Stats != nil {
			m.stats = f.Stats
		}
		return m, nil
	}
	// Legacy bare map: {logical: url}
	legacy := make(map[string]string, len(probe))
	for k, raw := range probe {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, fmt.Errorf("assets: parse manifest: legacy entry %q is not a string", k)
		}
		legacy[k] = s
	}
	m.entries = legacy
	return m, nil
}

// Save writes the manifest to a JSON file atomically. It writes to
// path+".tmp" then renames to path, so a crash mid-write leaves the
// previous manifest intact.
func (m *Manifest) Save(path string) error {
	m.mu.RLock()
	f := manifestFile{
		Assets: cloneStringMap(m.entries),
	}
	if len(m.sources) > 0 {
		f.Sources = cloneStringMap(m.sources)
	}
	if len(m.stats) > 0 {
		f.Stats = make(map[string]BuildStats, len(m.stats))
		for k, v := range m.stats {
			f.Stats[k] = v
		}
	}
	m.mu.RUnlock()

	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("assets: marshal manifest: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("assets: create manifest dir: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("assets: write manifest tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("assets: rename manifest: %w", err)
	}
	return nil
}

func cloneStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
