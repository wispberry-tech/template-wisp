package assets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManifest_ResolveHitMiss(t *testing.T) {
	m := NewManifest()
	m.Set("a.css", "/dist/a.abcd1234.css")

	url, ok := m.Resolve("a.css")
	require.True(t, ok)
	require.Equal(t, "/dist/a.abcd1234.css", url)

	_, ok = m.Resolve("missing.css")
	require.False(t, ok)
}

func TestManifest_EntriesIsCopy(t *testing.T) {
	m := NewManifest()
	m.Set("a.css", "/dist/a.css")
	entries := m.Entries()
	entries["mutated"] = "x"
	_, ok := m.Resolve("mutated")
	require.False(t, ok, "mutating returned map must not affect manifest")
}

func TestManifest_SaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	m := NewManifest()
	m.Set("a.css", "/dist/a.abcd1234.css")
	m.Set("b.js", "/dist/b.ef567890.js")
	m.SetSource("a.css", "/dist/a.abcd1234.css.map")
	m.SetStats("a.css", BuildStats{DurationMs: 5, InputBytes: 100, OutputBytes: 50, Ratio: 0.5})

	require.NoError(t, m.Save(path))

	loaded, err := LoadManifest(path)
	require.NoError(t, err)
	require.Equal(t, m.Entries(), loaded.Entries())
	require.Equal(t, m.Sources(), loaded.Sources())
	require.Equal(t, m.Stats(), loaded.Stats())
}

func TestManifest_SaveAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	// Seed with valid existing manifest.
	orig := NewManifest()
	orig.Set("a.css", "/dist/a.111.css")
	require.NoError(t, orig.Save(path))

	// Verify .tmp is cleaned up after successful save.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		require.NotEqual(t, "manifest.json.tmp", e.Name(), "tmp file leaked")
	}

	// Rewrite and confirm atomicity: read result matches latest.
	m2 := NewManifest()
	m2.Set("a.css", "/dist/a.222.css")
	require.NoError(t, m2.Save(path))

	loaded, err := LoadManifest(path)
	require.NoError(t, err)
	require.Equal(t, "/dist/a.222.css", loaded.Entries()["a.css"])
}

func TestManifest_LoadLegacyBareMap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.json")
	legacy := `{
  "a.css": "/dist/a.abcd1234.css",
  "b.js":  "/dist/b.ef567890.js"
}`
	require.NoError(t, os.WriteFile(path, []byte(legacy), 0o644))

	loaded, err := LoadManifest(path)
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"a.css": "/dist/a.abcd1234.css",
		"b.js":  "/dist/b.ef567890.js",
	}, loaded.Entries())
	require.Empty(t, loaded.Sources())
	require.Empty(t, loaded.Stats())
}

func TestManifest_LoadMissingFile(t *testing.T) {
	_, err := LoadManifest("/does/not/exist/manifest.json")
	require.Error(t, err)
}

func TestManifest_Delete(t *testing.T) {
	m := NewManifest()
	m.Set("a.css", "/dist/a.css")
	m.SetSource("a.css", "/dist/a.css.map")
	m.SetStats("a.css", BuildStats{InputBytes: 10})

	m.Delete("a.css")
	_, ok := m.Resolve("a.css")
	require.False(t, ok)
	require.Empty(t, m.Sources())
	require.Empty(t, m.Stats())
}

func TestNoopTransformer(t *testing.T) {
	tr := NoopTransformer{}
	in := []byte(".foo { color: red; }")
	out, err := tr.Transform(in, "text/css")
	require.NoError(t, err)
	require.Equal(t, in, out)
}
