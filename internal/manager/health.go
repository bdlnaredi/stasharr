package manager

// The Health page: disk space, the database and backups, the tagger,
// auto-import and recent failed tasks, in one place, with a notification
// when a disk runs low.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/sqlite"
)

func fmtUint(v uint64) string { return strconv.FormatUint(v, 10) }

// Disk is one disk and the folders on it.
type Disk struct {
	Labels []string
	Paths  []string
	Total  uint64
	Free   uint64
	Low    bool
}

// diskIsLow is free space under the limit in GB, or under 5%.
func diskIsLow(total, free uint64) bool {
	minFree := uint64(config.GetInstance().GetHealthMinFreeGB()) * 1_000_000_000
	return free < minFree || (total > 0 && float64(free)/float64(total) < 0.05)
}

// Disks lists the disks holding the library, generated files, the
// database and backups.
func (s *Manager) Disks() []Disk {
	cfg := s.Config
	type place struct{ label, path string }
	var places []place
	for _, st := range cfg.GetStashPaths() {
		places = append(places, place{"Library", st.Path})
	}
	places = append(places,
		place{"Generated files", cfg.GetGeneratedPath()},
		place{"Database", filepath.Dir(cfg.GetDatabasePath())},
		place{"Backups", cfg.GetBackupDirectoryPathOrDefault()},
		place{"Cache", cfg.GetCachePath()},
	)
	byID := map[string]*Disk{}
	var order []string
	for _, p := range places {
		if p.path == "" {
			continue
		}
		total, free, id, err := diskSpace(p.path)
		if err != nil {
			continue
		}
		if id == "" {
			id = fmt.Sprintf("%d/%d", total, free)
		}
		d := byID[id]
		if d == nil {
			d = &Disk{Total: total, Free: free, Low: diskIsLow(total, free)}
			byID[id] = d
			order = append(order, id)
		}
		if !containsString(d.Labels, p.label) {
			d.Labels = append(d.Labels, p.label)
		}
		d.Paths = append(d.Paths, p.path)
	}
	ret := make([]Disk, 0, len(order))
	for _, id := range order {
		ret = append(ret, *byID[id])
	}
	return ret
}

func containsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------- failed tasks

// FailedTask is a task that failed recently.
type FailedTask struct {
	Description string
	Error       string
	At          time.Time
}

var failures struct {
	mu   sync.Mutex
	list []FailedTask
}

func rememberFailure(description, err string) {
	failures.mu.Lock()
	defer failures.mu.Unlock()
	failures.list = append([]FailedTask{{Description: description, Error: err, At: time.Now()}}, failures.list...)
	if len(failures.list) > 20 {
		failures.list = failures.list[:20]
	}
}

// RecentFailures returns tasks that failed since the server started, newest
// first.
func RecentFailures() []FailedTask {
	failures.mu.Lock()
	defer failures.mu.Unlock()
	return append([]FailedTask(nil), failures.list...)
}

func init() {
	prev := job.OnJobFailed
	job.OnJobFailed = func(description, err string) {
		rememberFailure(description, err)
		if prev != nil {
			prev(description, err)
		}
	}
}

// ---------------------------------------------------------------- alerts

var lowDiskWarned = map[string]time.Time{}

// healthTick runs every 10 minutes: a notification when a disk runs low
// (at most every 12 hours per disk), and background upkeep.
func (s *Manager) healthTick() {
	for _, d := range s.Disks() {
		key := fmt.Sprint(d.Paths)
		if !d.Low {
			delete(lowDiskWarned, key)
			continue
		}
		if t, ok := lowDiskWarned[key]; ok && time.Since(t) < 12*time.Hour {
			continue
		}
		lowDiskWarned[key] = time.Now()
		msg := fmt.Sprintf("%s: %s free of %s (%s).", joinLabels(d.Labels), HumanBytes(d.Free), HumanBytes(d.Total), d.Paths[0])
		logger.Warnf("[health] low disk space: %s", msg)
		Notify(EventHealth, "Low disk space", msg)
	}
	s.backfillSceneMeans()
}

func joinLabels(l []string) string {
	sort.Strings(l)
	out := ""
	for i, x := range l {
		if i > 0 {
			out += ", "
		}
		out += x
	}
	return out
}

// HumanBytes formats a size like 12.3 GB.
func HumanBytes(b uint64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}

// backfillSceneMeans gives scenes indexed before taste recommendations
// their mean vector, a few hundred at a time.
func (s *Manager) backfillSceneMeans() {
	ctx := context.Background()
	for i := 0; i < 20; i++ {
		n := 0
		if err := s.Repository.WithTxn(ctx, func(ctx context.Context) error {
			var err error
			n, err = sqlite.BackfillSceneMeans(ctx, 200)
			return err
		}); err != nil {
			logger.Warnf("[recommend] %v", err)
			return
		}
		if n == 0 {
			return
		}
	}
}

// DatabaseSize is the size of the database file.
func (s *Manager) DatabaseSize() int64 {
	st, err := os.Stat(s.Config.GetDatabasePath())
	if err != nil {
		return 0
	}
	return st.Size()
}
