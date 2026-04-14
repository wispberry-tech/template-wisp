package assets

import (
	"context"
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

// watchPollInterval is how often the watcher polls SourceDir for mtime changes.
const watchPollInterval = 500 * time.Millisecond

// watchDebounce collects rapid consecutive changes before rebuilding.
const watchDebounce = 100 * time.Millisecond

// Watch performs an initial Build(), then watches SourceDir for file changes
// via mtime polling. On detected change, it rebuilds only changed files,
// preserving manifest entries for any that fail (partial swap), then calls
// OnChange with the updated manifest.
//
// Watch blocks until ctx is cancelled. Returns the initial-build error if
// the first Build() fails, otherwise nil after ctx.Done().
func (b *Builder) Watch(ctx context.Context, h WatchHandlers) error {
	if h.OnChange == nil {
		return errors.New("assets: WatchHandlers.OnChange is required")
	}

	// Initial full build.
	manifest, err := b.build(h.OnEvent)
	if err != nil {
		return err
	}
	h.OnChange(manifest)

	// Seed mtime cache so we only rebuild on subsequent changes.
	mtimes, err := b.scanMTimes()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(watchPollInterval)
	defer ticker.Stop()

	var (
		pendingMu sync.Mutex
		pending   = make(map[string]struct{}) // changed paths awaiting rebuild
		lastSeen  time.Time
	)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			current, scanErr := b.scanMTimes()
			if scanErr != nil {
				if h.OnError != nil {
					h.OnError(scanErr)
				}
				continue
			}
			// Collect changes.
			changed := diffMTimes(mtimes, current)
			if len(changed) > 0 {
				pendingMu.Lock()
				for _, p := range changed {
					pending[p] = struct{}{}
				}
				lastSeen = time.Now()
				pendingMu.Unlock()
			}
			mtimes = current

			// Debounce: only rebuild when quiet period elapsed.
			pendingMu.Lock()
			shouldRebuild := len(pending) > 0 && time.Since(lastSeen) >= watchDebounce
			var toRebuild map[string]struct{}
			if shouldRebuild {
				toRebuild = pending
				pending = make(map[string]struct{})
			}
			pendingMu.Unlock()

			if !shouldRebuild {
				continue
			}

			newManifest, failures := b.partialRebuild(manifest, toRebuild, h.OnEvent)
			for _, fe := range failures {
				if h.OnError != nil {
					h.OnError(fe)
				}
			}
			manifest = newManifest
			h.OnChange(manifest)
		}
	}
}

// scanMTimes walks SourceDir and returns a path->mtime map for every file
// matching configured extensions.
func (b *Builder) scanMTimes() (map[string]time.Time, error) {
	exts := extSet(b.cfg.Extensions)
	out := make(map[string]time.Time)
	err := filepath.WalkDir(b.cfg.SourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if _, ok := exts[strings.ToLower(filepath.Ext(path))]; !ok {
			return nil
		}
		info, iErr := d.Info()
		if iErr != nil {
			return iErr
		}
		out[path] = info.ModTime()
		return nil
	})
	return out, err
}

// diffMTimes returns paths that were added, removed, or changed between old and new.
func diffMTimes(old, new map[string]time.Time) []string {
	var changed []string
	for p, newT := range new {
		if oldT, ok := old[p]; !ok || !oldT.Equal(newT) {
			changed = append(changed, p)
		}
	}
	for p := range old {
		if _, ok := new[p]; !ok {
			changed = append(changed, p)
		}
	}
	return changed
}

// partialRebuild applies targeted rebuilds for the given source paths,
// preserving entries from prev for files that fail. Returns the new manifest
// and the list of per-file errors.
func (b *Builder) partialRebuild(prev *Manifest, paths map[string]struct{}, onEvent func(Event)) (*Manifest, []error) {
	b.buildMu.Lock()
	defer b.buildMu.Unlock()

	// Start from prev's current state.
	next := NewManifest()
	for k, v := range prev.Entries() {
		next.Set(k, v)
	}
	for k, v := range prev.Sources() {
		next.SetSource(k, v)
	}
	for k, v := range prev.Stats() {
		next.SetStats(k, v)
	}

	var failures []error

	for path := range paths {
		rel, err := filepath.Rel(b.cfg.SourceDir, path)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		logical := filepath.ToSlash(rel)

		// If the source file is gone, drop the manifest entry.
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			next.Delete(logical)
			emit(onEvent, Event{Type: EventPruned, LogicalName: logical})
			continue
		}

		ext := strings.ToLower(filepath.Ext(path))
		if err := b.rebuildOne(path, logical, ext, next, onEvent); err != nil {
			// Partial swap: keep prev entry (already copied above).
			failures = append(failures, fmt.Errorf("assets: rebuild %q: %w", logical, err))
		}
	}

	// Prune pass (if enabled).
	if b.cfg.PruneUnreferenced && b.refProvider != nil {
		refs := b.refProvider()
		if len(refs) > 0 {
			for logical := range next.Entries() {
				if _, used := refs[logical]; !used {
					next.Delete(logical)
					emit(onEvent, Event{Type: EventPruned, LogicalName: logical})
				}
			}
		}
	}

	// Persist manifest if configured.
	if b.cfg.ManifestPath != "" {
		if err := next.Save(b.cfg.ManifestPath); err != nil {
			failures = append(failures, err)
		}
	}

	return next, failures
}

// rebuildOne processes a single file during a watch-mode partial rebuild.
// Does NOT take buildMu (caller already holds it).
func (b *Builder) rebuildOne(srcPath, logical, ext string, manifest *Manifest, onEvent func(Event)) error {
	start := time.Now()
	raw, err := os.ReadFile(srcPath)
	if err != nil {
		emit(onEvent, Event{Type: EventError, LogicalName: logical, Err: err})
		return err
	}

	tr, mediaType := b.transformerFor(ext)
	out, err := tr.Transform(raw, mediaType)
	if err != nil {
		emit(onEvent, Event{Type: EventError, LogicalName: logical, Err: err})
		return err
	}

	var hashPart string
	if b.cfg.HashFiles {
		sum := sha256.Sum256(out)
		hashPart = hex.EncodeToString(sum[:4])
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
