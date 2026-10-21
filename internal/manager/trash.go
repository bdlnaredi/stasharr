package manager

// Stasharr trash bin: deleted library files are moved to a trash folder and
// deleted for good after the retention period.

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/file"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/trash"
)

// setupTrash points file deletion at the trash bin and starts the purge loop.
func (s *Manager) setupTrash() {
	cfg := s.Config
	trash.Configure(filepath.Join(cfg.GetConfigPath(), "stasharr-trash.json"), cfg.GetDeleteTrashPath)

	file.TrashDirFor = func(path string) string {
		c := config.GetInstance()
		if !c.GetTrashEnabled() {
			return ""
		}
		root := c.GetStashPaths().GetStashRootFromDirPath(filepath.Dir(path))
		if root == "" {
			// not a library file, for example a generated file
			return ""
		}
		return filepath.Join(root, trash.DirName)
	}

	if !s.trashLoopStarted {
		s.trashLoopStarted = true
		go trashPurgeLoop()
	}
}

func trashPurgeLoop() {
	// first purge shortly after start
	wait := 5 * time.Minute
	for {
		time.Sleep(wait)
		wait = time.Hour
		PurgeTrash()
	}
}

// PurgeTrash deletes trash bin entries older than the retention period.
func PurgeTrash() int {
	days := config.GetInstance().GetTrashRetentionDays()
	if days <= 0 {
		return 0
	}
	return trash.Purge(time.Duration(days)*24*time.Hour, TrashDirs())
}

// TrashDirs returns every trash folder in use.
func TrashDirs() []string {
	cfg := config.GetInstance()
	var ret []string
	if p := cfg.GetDeleteTrashPath(); p != "" {
		ret = append(ret, p)
	}
	for _, s := range cfg.GetStashPaths() {
		ret = append(ret, filepath.Join(s.Path, trash.DirName))
	}
	return ret
}

// RestoreFromTrash moves a file back to its library folder and imports it
// again: scan, identify and generate, like a new download.
func RestoreFromTrash(ctx context.Context, id string) (string, error) {
	e, err := trash.Restore(id)
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(e.OriginalPath)
	if e.IsDir {
		dir = e.OriginalPath
	}

	mgr := GetInstance()
	if mgr.Watcher != nil {
		if err := mgr.Watcher.ImportNow([]string{dir}); err == nil {
			return e.OriginalPath, nil
		} else if !errors.Is(err, errWatcherBusy) {
			logger.Warnf("[trash] %v", err)
		}
	}

	// the watcher is busy: a plain scan still brings the file back
	input := ScanMetadataInput{Paths: []string{dir}}
	input.ScanGenerateCovers = true
	input.ScanGeneratePhashes = true
	if _, err := mgr.Scan(ctx, input); err != nil {
		return e.OriginalPath, err
	}
	return e.OriginalPath, nil
}
