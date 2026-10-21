// Package trash keeps the Stasharr trash bin: an index of files that were
// deleted from the library and moved to a trash folder instead of being
// removed straight away. Entries can be restored or are purged after the
// retention period.
//
// The index is a JSON file in the config folder. Trash folders are either the
// upstream "delete trash path" setting or, when that is empty, a folder named
// DirName at the top of each library folder, so moving a file to the trash is
// an instant rename on the same disk.
package trash

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/stashapp/stash/pkg/logger"
)

// DirName is the trash folder created inside library folders.
const DirName = ".stasharr-trash"

// Entry is one deleted file or folder.
type Entry struct {
	ID string `json:"id"`
	// OriginalPath is where the file was before it was deleted. Empty for
	// files found in a trash folder that were not deleted by Stasharr.
	OriginalPath string    `json:"original_path"`
	TrashPath    string    `json:"trash_path"`
	Size         int64     `json:"size"`
	IsDir        bool      `json:"is_dir"`
	DeletedAt    time.Time `json:"deleted_at"`
}

var (
	mu        sync.Mutex
	indexFile string
	// configuredPath returns the "delete trash path" setting, if any
	configuredPath func() string
)

// Configure sets the index file and a function returning the configured
// trash folder, if any.
func Configure(index string, configuredTrashPath func() string) {
	mu.Lock()
	defer mu.Unlock()
	indexFile = index
	configuredPath = configuredTrashPath
}

// IsTrashPath reports whether path is a trash folder or inside one, so that
// scans and the library watcher leave trashed files alone.
func IsTrashPath(path string) bool {
	clean := filepath.Clean(path)
	for _, part := range strings.FieldsFunc(clean, func(r rune) bool { return r == '/' || r == '\\' }) {
		if part == DirName {
			return true
		}
	}
	mu.Lock()
	fn := configuredPath
	mu.Unlock()
	if fn == nil {
		return false
	}
	cp := strings.TrimRight(fn(), `/\`)
	if cp == "" {
		return false
	}
	cp = filepath.Clean(cp)
	return clean == cp || strings.HasPrefix(clean, cp+string(filepath.Separator))
}

func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// load must be called with mu held.
func load() ([]Entry, error) {
	if indexFile == "" {
		return nil, errors.New("trash bin is not ready yet")
	}
	data, err := os.ReadFile(indexFile)
	if errors.Is(err, fs.ErrNotExist) {
		return []Entry{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading trash index: %w", err)
	}
	var ret []Entry
	if len(data) == 0 {
		return []Entry{}, nil
	}
	if err := json.Unmarshal(data, &ret); err != nil {
		return nil, fmt.Errorf("reading trash index: %w", err)
	}
	return ret, nil
}

// save must be called with mu held.
func save(entries []Entry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	tmp := indexFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("writing trash index: %w", err)
	}
	return os.Rename(tmp, indexFile)
}

func sizeOf(path string) (int64, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, false
	}
	if !info.IsDir() {
		return info.Size(), false
	}
	var total int64
	_ = filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if i, err := d.Info(); err == nil {
			total += i.Size()
		}
		return nil
	})
	return total, true
}

// Record adds deleted files to the index. trashed maps original path to the
// path in the trash.
func Record(trashed map[string]string) {
	if len(trashed) == 0 {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	entries, err := load()
	if err != nil {
		logger.Warnf("[trash] %v", err)
		return
	}
	now := time.Now()
	for orig, t := range trashed {
		size, isDir := sizeOf(t)
		entries = append(entries, Entry{
			ID:           newID(),
			OriginalPath: orig,
			TrashPath:    t,
			Size:         size,
			IsDir:        isDir,
			DeletedAt:    now,
		})
	}
	if err := save(entries); err != nil {
		logger.Warnf("[trash] %v", err)
	}
}

// List returns the trash contents, newest first. Files found in the given
// trash folders that are not in the index yet are added, so their retention
// period starts now. Entries whose file is gone are dropped.
func List(trashDirs []string) ([]Entry, error) {
	mu.Lock()
	defer mu.Unlock()
	entries, err := load()
	if err != nil {
		return nil, err
	}

	known := map[string]bool{}
	kept := entries[:0]
	changed := false
	for _, e := range entries {
		if _, err := os.Lstat(e.TrashPath); err != nil {
			changed = true
			continue
		}
		known[filepath.Clean(e.TrashPath)] = true
		kept = append(kept, e)
	}
	entries = kept

	now := time.Now()
	for _, dir := range trashDirs {
		items, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, it := range items {
			p := filepath.Clean(filepath.Join(dir, it.Name()))
			if known[p] {
				continue
			}
			size, isDir := sizeOf(p)
			entries = append(entries, Entry{ID: newID(), TrashPath: p, Size: size, IsDir: isDir, DeletedAt: now})
			known[p] = true
			changed = true
		}
	}

	if changed {
		if err := save(entries); err != nil {
			return nil, err
		}
	}

	ret := make([]Entry, len(entries))
	copy(ret, entries)
	sort.SliceStable(ret, func(i, j int) bool { return ret[i].DeletedAt.After(ret[j].DeletedAt) })
	return ret, nil
}

// Restore moves an entry back to where it was and removes it from the index.
func Restore(id string) (Entry, error) {
	mu.Lock()
	defer mu.Unlock()
	entries, err := load()
	if err != nil {
		return Entry{}, err
	}
	for i, e := range entries {
		if e.ID != id {
			continue
		}
		if e.OriginalPath == "" {
			return e, errors.New("the original location of this file is not known")
		}
		if _, err := os.Lstat(e.OriginalPath); err == nil {
			return e, fmt.Errorf("%s already exists", e.OriginalPath)
		}
		if err := os.MkdirAll(filepath.Dir(e.OriginalPath), 0755); err != nil {
			return e, fmt.Errorf("creating folder: %w", err)
		}
		if err := move(e.TrashPath, e.OriginalPath); err != nil {
			return e, err
		}
		entries = append(entries[:i], entries[i+1:]...)
		if err := save(entries); err != nil {
			logger.Warnf("[trash] %v", err)
		}
		logger.Infof("[trash] restored %s", e.OriginalPath)
		return e, nil
	}
	return Entry{}, errors.New("not in the trash bin")
}

func move(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	// different disk: copy then remove
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("cannot move folder %s to another disk", src)
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode())
	if err != nil {
		return err
	}
	if _, err := out.ReadFrom(in); err != nil {
		out.Close()
		_ = os.Remove(dst)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(dst)
		return err
	}
	_ = os.Chtimes(dst, time.Now(), info.ModTime())
	in.Close()
	return os.Remove(src)
}

// Delete removes entries from disk permanently. An empty ids list empties the
// whole trash bin. It returns the number removed.
func Delete(ids []string) (int, error) {
	mu.Lock()
	defer mu.Unlock()
	entries, err := load()
	if err != nil {
		return 0, err
	}
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	removed := 0
	kept := entries[:0]
	for _, e := range entries {
		if len(ids) > 0 && !want[e.ID] {
			kept = append(kept, e)
			continue
		}
		if err := os.RemoveAll(e.TrashPath); err != nil {
			logger.Warnf("[trash] could not delete %s: %v", e.TrashPath, err)
			kept = append(kept, e)
			continue
		}
		removed++
	}
	if err := save(kept); err != nil {
		return removed, err
	}
	return removed, nil
}

// Purge permanently deletes entries older than the retention period.
func Purge(retention time.Duration, trashDirs []string) int {
	if retention <= 0 {
		return 0
	}
	// pick up files that were put in the trash some other way
	if _, err := List(trashDirs); err != nil {
		logger.Warnf("[trash] %v", err)
		return 0
	}

	mu.Lock()
	entries, err := load()
	mu.Unlock()
	if err != nil {
		logger.Warnf("[trash] %v", err)
		return 0
	}
	cutoff := time.Now().Add(-retention)
	var ids []string
	for _, e := range entries {
		if e.DeletedAt.Before(cutoff) {
			ids = append(ids, e.ID)
		}
	}
	if len(ids) == 0 {
		return 0
	}
	n, err := Delete(ids)
	if err != nil {
		logger.Warnf("[trash] %v", err)
	}
	if n > 0 {
		logger.Infof("[trash] permanently deleted %d item(s) older than %d days", n, int(retention.Hours()/24))
	}
	return n
}
