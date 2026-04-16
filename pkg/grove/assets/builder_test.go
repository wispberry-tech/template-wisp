package assets

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// writeTree writes files under root. keys are forward-slash paths, values are contents.
func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	}
}

func TestBuilder_WritesHashedFiles(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeTree(t, src, map[string]string{
		"a.css":           ".a { color: red; }",
		"primitives/b.js": "console.log('b');",
		"ignored.txt":     "skip me",
	})

	b := NewWithDefaults(Config{SourceDir: src, OutputDir: dst})
	m, err := b.Build()
	require.NoError(t, err)

	entries := m.Entries()
	require.Len(t, entries, 2, "only .css and .js should be included")

	// Check files exist in OutputDir with hash in name.
	aURL := entries["a.css"]
	require.Regexp(t, `^/dist/a\.[0-9a-f]{8}\.css$`, aURL)
	bURL := entries["primitives/b.js"]
	require.Regexp(t, `^/dist/primitives/b\.[0-9a-f]{8}\.js$`, bURL)

	// Verify file on disk.
	diskPath := filepath.Join(dst, strings.TrimPrefix(aURL, "/dist/"))
	data, err := os.ReadFile(diskPath)
	require.NoError(t, err)
	require.Equal(t, ".a { color: red; }", string(data))
}

func TestBuilder_HashStability(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeTree(t, src, map[string]string{"a.css": ".a{}"})

	b := NewWithDefaults(Config{SourceDir: src, OutputDir: dst})
	m1, err := b.Build()
	require.NoError(t, err)

	m2, err := b.Build()
	require.NoError(t, err)
	require.Equal(t, m1.Entries(), m2.Entries(), "same content → same hash")
}

func TestBuilder_HashChangesOnContentChange(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeTree(t, src, map[string]string{"a.css": ".a{}"})

	b := NewWithDefaults(Config{SourceDir: src, OutputDir: dst})
	m1, err := b.Build()
	require.NoError(t, err)

	writeTree(t, src, map[string]string{"a.css": ".b{}"})
	m2, err := b.Build()
	require.NoError(t, err)

	require.NotEqual(t, m1.Entries()["a.css"], m2.Entries()["a.css"])
}

func TestBuilder_HashFilesFalse(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeTree(t, src, map[string]string{"a.css": ".a{}"})

	b := New(Config{SourceDir: src, OutputDir: dst, HashFiles: false})
	m, err := b.Build()
	require.NoError(t, err)
	require.Equal(t, "/dist/a.css", m.Entries()["a.css"])
}

func TestBuilder_ManifestSavedToDisk(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeTree(t, src, map[string]string{"a.css": ".a{}"})

	manifestPath := filepath.Join(dst, "manifest.json")
	b := NewWithDefaults(Config{
		SourceDir:    src,
		OutputDir:    dst,
		ManifestPath: manifestPath,
	})
	_, err := b.Build()
	require.NoError(t, err)

	loaded, err := LoadManifest(manifestPath)
	require.NoError(t, err)
	require.Contains(t, loaded.Entries(), "a.css")
}

func TestBuilder_Prune(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeTree(t, src, map[string]string{
		"used.css":   ".u{}",
		"unused.css": ".x{}",
	})

	b := NewWithDefaults(Config{
		SourceDir:         src,
		OutputDir:         dst,
		PruneUnreferenced: true,
	})
	b.SetReferencedNameProvider(func() map[string]struct{} {
		return map[string]struct{}{"used.css": {}}
	})

	m, err := b.Build()
	require.NoError(t, err)
	require.Contains(t, m.Entries(), "used.css")
	require.NotContains(t, m.Entries(), "unused.css")
}

func TestBuilder_PruneSkippedWhenProviderNil(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeTree(t, src, map[string]string{"a.css": ".a{}", "b.css": ".b{}"})

	b := NewWithDefaults(Config{SourceDir: src, OutputDir: dst, PruneUnreferenced: true})
	m, err := b.Build()
	require.NoError(t, err)
	require.Len(t, m.Entries(), 2, "no provider → no prune")
}

func TestBuilder_PruneSkippedWhenRefsEmpty(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeTree(t, src, map[string]string{"a.css": ".a{}"})

	b := NewWithDefaults(Config{SourceDir: src, OutputDir: dst, PruneUnreferenced: true})
	b.SetReferencedNameProvider(func() map[string]struct{} { return nil })

	m, err := b.Build()
	require.NoError(t, err)
	require.Len(t, m.Entries(), 1, "empty refs → no prune (first build semantics)")
}

func TestBuilder_CustomExtensions(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeTree(t, src, map[string]string{
		"a.css":  ".a{}",
		"b.mjs":  "export{}",
		"c.scss": ".c{}",
	})

	b := NewWithDefaults(Config{
		SourceDir:  src,
		OutputDir:  dst,
		Extensions: []string{".mjs", ".scss"},
	})
	m, err := b.Build()
	require.NoError(t, err)
	require.Len(t, m.Entries(), 2)
	require.Contains(t, m.Entries(), "b.mjs")
	require.Contains(t, m.Entries(), "c.scss")
}

func TestBuilder_ConcurrentBuildSerializes(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeTree(t, src, map[string]string{"a.css": ".a{}"})

	b := NewWithDefaults(Config{SourceDir: src, OutputDir: dst})

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := b.Build()
			require.NoError(t, err)
		}()
	}
	wg.Wait()
}

func TestBuilder_IncludeBuildStats(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeTree(t, src, map[string]string{"a.css": ".a { color: red; }"})

	b := NewWithDefaults(Config{
		SourceDir:         src,
		OutputDir:         dst,
		IncludeBuildStats: true,
	})
	m, err := b.Build()
	require.NoError(t, err)

	stats := m.Stats()["a.css"]
	require.Equal(t, 18, stats.InputBytes)
	require.Greater(t, stats.OutputBytes, 0)
}

func TestBuilder_EmitsEvents(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeTree(t, src, map[string]string{"a.css": ".a{}"})

	b := NewWithDefaults(Config{SourceDir: src, OutputDir: dst})

	var mu sync.Mutex
	types := []EventType{}
	_, err := b.build(func(e Event) {
		mu.Lock()
		types = append(types, e.Type)
		mu.Unlock()
	})
	require.NoError(t, err)
	require.Contains(t, types, EventDiscovered)
	require.Contains(t, types, EventBuilt)
}

func TestBuilder_MissingSourceDir(t *testing.T) {
	b := NewWithDefaults(Config{OutputDir: t.TempDir()})
	_, err := b.Build()
	require.Error(t, err)
}

func TestBuilder_MissingOutputDir(t *testing.T) {
	b := NewWithDefaults(Config{SourceDir: t.TempDir()})
	_, err := b.Build()
	require.Error(t, err)
}
