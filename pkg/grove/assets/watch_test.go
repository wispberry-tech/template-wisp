package assets

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWatch_InitialBuildFires(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeTree(t, src, map[string]string{"a.css": ".a{}"})

	b := NewWithDefaults(Config{SourceDir: src, OutputDir: dst})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	var got atomic.Int32

	go func() {
		err := b.Watch(ctx, WatchHandlers{
			OnChange: func(m *Manifest) {
				got.Add(1)
				if _, ok := m.Resolve("a.css"); ok {
					select {
					case <-done:
					default:
						close(done)
					}
				}
			},
		})
		require.NoError(t, err)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("watch did not fire initial OnChange")
	}
	require.GreaterOrEqual(t, got.Load(), int32(1))
}

func TestWatch_FileChangeTriggersRebuild(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeTree(t, src, map[string]string{"a.css": ".a{}"})

	b := NewWithDefaults(Config{SourceDir: src, OutputDir: dst})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	manifests := []map[string]string{}
	calls := make(chan struct{}, 16)

	go func() {
		_ = b.Watch(ctx, WatchHandlers{
			OnChange: func(m *Manifest) {
				mu.Lock()
				manifests = append(manifests, m.Entries())
				mu.Unlock()
				select {
				case calls <- struct{}{}:
				default:
				}
			},
		})
	}()

	// Wait for initial build.
	select {
	case <-calls:
	case <-time.After(2 * time.Second):
		t.Fatal("no initial OnChange")
	}

	// Ensure mtime actually changes on the next write.
	time.Sleep(1100 * time.Millisecond)
	require.NoError(t, os.WriteFile(filepath.Join(src, "a.css"), []byte(".aa{}"), 0o644))

	// Wait for rebuild.
	select {
	case <-calls:
	case <-time.After(3 * time.Second):
		t.Fatal("no rebuild after file change")
	}

	mu.Lock()
	defer mu.Unlock()
	require.GreaterOrEqual(t, len(manifests), 2)
	require.NotEqual(t, manifests[0]["a.css"], manifests[len(manifests)-1]["a.css"])
}

// failingTransformer wraps NoopTransformer but errors on files containing "BAD".
type failingTransformer struct{}

func (failingTransformer) Transform(src []byte, _ string) ([]byte, error) {
	if len(src) >= 3 && string(src[:3]) == "BAD" {
		return nil, errors.New("deliberate failure")
	}
	return src, nil
}

func TestWatch_PartialSwapOnFailure(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeTree(t, src, map[string]string{
		"good.css": ".good{}",
		"bad.css":  ".bad{}",
	})

	b := NewWithDefaults(Config{
		SourceDir:      src,
		OutputDir:      dst,
		CSSTransformer: failingTransformer{},
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type snap struct {
		entries  map[string]string
		failures int
	}
	ch := make(chan snap, 16)
	var failures atomic.Int32

	go func() {
		_ = b.Watch(ctx, WatchHandlers{
			OnChange: func(m *Manifest) {
				ch <- snap{entries: m.Entries(), failures: int(failures.Load())}
			},
			OnError: func(error) {
				failures.Add(1)
			},
		})
	}()

	// Initial build: both files good.
	select {
	case first := <-ch:
		require.Contains(t, first.entries, "good.css")
		require.Contains(t, first.entries, "bad.css")
	case <-time.After(2 * time.Second):
		t.Fatal("no initial OnChange")
	}

	goodURL := ""
	select {
	case first := <-ch:
		goodURL = first.entries["good.css"]
	default:
	}
	// Capture current good URL.
	if goodURL == "" {
		m, err := b.Build()
		require.NoError(t, err)
		goodURL = m.Entries()["good.css"]
	}

	// Make bad.css fail AND change good.css.
	time.Sleep(1100 * time.Millisecond)
	require.NoError(t, os.WriteFile(filepath.Join(src, "bad.css"), []byte("BAD bad bad"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "good.css"), []byte(".good2{}"), 0o644))

	// Wait for rebuild with failure.
	deadline := time.After(4 * time.Second)
	for {
		select {
		case s := <-ch:
			if s.failures > 0 {
				// good.css should have updated URL, bad.css kept prior.
				require.NotEqual(t, goodURL, s.entries["good.css"])
				require.Contains(t, s.entries, "bad.css", "bad entry should be retained")
				return
			}
		case <-deadline:
			t.Fatal("no rebuild with failure observed")
		}
	}
}

func TestWatch_CtxCancelStops(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeTree(t, src, map[string]string{"a.css": ".a{}"})

	b := NewWithDefaults(Config{SourceDir: src, OutputDir: dst})
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- b.Watch(ctx, WatchHandlers{OnChange: func(*Manifest) {}})
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("watch did not exit after cancel")
	}
}

func TestWatch_InitialBuildFailureReturns(t *testing.T) {
	// SourceDir missing → initial build fails.
	b := NewWithDefaults(Config{SourceDir: "/nope/missing", OutputDir: t.TempDir()})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := b.Watch(ctx, WatchHandlers{OnChange: func(*Manifest) {}})
	require.Error(t, err)
}

func TestWatch_RequiresOnChange(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	b := NewWithDefaults(Config{SourceDir: src, OutputDir: dst})
	err := b.Watch(context.Background(), WatchHandlers{})
	require.Error(t, err)
}
