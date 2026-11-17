package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/stashapp/stash/internal/build"
	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sqlite"
)

func (r *queryResolver) StasharrHealth(ctx context.Context) (*HealthReport, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	mgr := manager.GetInstance()
	cfg := config.GetInstance()
	version, _, _ := build.Version()
	ret := &HealthReport{
		MinFreeGb:      cfg.GetHealthMinFreeGB(),
		DatabaseSize:   mgr.DatabaseSize(),
		BackupsEnabled: cfg.GetBackupScheduleEnabled(),
		DriveConnected: cfg.GetGDriveRefreshToken() != "",
		TaggerSet:      cfg.GetAIURL() != "",
		WatcherEnabled: cfg.GetWatcherEnabled(),
		NotifyTargets:  0,
		Version:        version,
		Disks:          []*HealthDisk{},
		FailedTasks:    []*HealthFailedTask{},
		Problems:       []*HealthProblem{},
	}
	problem := func(level, format string, a ...any) {
		ret.Problems = append(ret.Problems, &HealthProblem{Level: level, Text: fmt.Sprintf(format, a...)})
	}

	for _, d := range mgr.Disks() {
		ret.Disks = append(ret.Disks, &HealthDisk{Labels: d.Labels, Paths: d.Paths, Total: int64(d.Total), Free: int64(d.Free), Low: d.Low})
		if d.Low {
			problem("error", "Low disk space for %s: %s free.", joinNames(d.Labels), manager.HumanBytes(d.Free))
		}
	}

	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		n, err := r.repository.Scene.Count(ctx)
		ret.Scenes = n
		return err
	}); err != nil {
		return nil, err
	}

	// backups
	if backups, err := mgr.ListBackups(); err == nil {
		for _, b := range backups {
			if ret.LastBackup == nil || b.CreatedAt.After(*ret.LastBackup) {
				t := b.CreatedAt
				ret.LastBackup = &t
			}
		}
	}
	switch {
	case !ret.BackupsEnabled:
		problem("warning", "Scheduled backups are off (Settings → Backups & refresh).")
	case ret.LastBackup == nil || time.Since(*ret.LastBackup) > 3*24*time.Hour:
		problem("warning", "No backup in the last 3 days.")
	}
	if e := cfg.GetGDriveLastError(); e != "" && ret.DriveConnected {
		ret.DriveLastError = &e
		problem("warning", "The last upload to Google Drive failed: %s", e)
	}

	// tagger
	if ret.TaggerSet {
		st, err := manager.GetTaggerStatus(ctx, cfg.GetAIURL())
		switch {
		case err != nil:
			d := err.Error()
			ret.TaggerDetail = &d
			problem("warning", "The Stasharr Tagger cannot be reached: %s", d)
		case !st.OK:
			d := "starting"
			if st.Error != nil {
				d = *st.Error
			}
			ret.TaggerDetail = &d
		default:
			ret.TaggerOk = true
			d := st.Model
			if st.GPU != nil && *st.GPU != "" {
				d += " on " + *st.GPU
			} else if st.Device != "" {
				d += " on " + st.Device
			}
			ret.TaggerDetail = &d
		}
	}

	// auto-import
	if mgr.Watcher != nil {
		ret.WatcherRunning = mgr.Watcher.Running()
		if at, _ := mgr.Watcher.Status(); !at.IsZero() {
			ret.WatcherLastCheck = &at
		}
	}
	if ret.WatcherEnabled && !ret.WatcherRunning {
		problem("warning", "Auto-import is on but not running.")
	}

	for _, t := range manager.GetNotifyTargets() {
		if t.Enabled {
			ret.NotifyTargets++
		}
	}
	if ret.NotifyTargets == 0 {
		problem("warning", "No notifications are set up, so you will not hear about problems (Settings → Notifications).")
	}

	for _, f := range manager.RecentFailures() {
		ret.FailedTasks = append(ret.FailedTasks, &HealthFailedTask{Description: f.Description, Error: f.Error, At: f.At})
	}
	if n := len(ret.FailedTasks); n > 0 && time.Since(ret.FailedTasks[0].At) < 24*time.Hour {
		problem("warning", "%d task(s) failed recently.", n)
	}
	return ret, nil
}

func joinNames(l []string) string {
	out := ""
	for i, x := range l {
		if i > 0 {
			out += ", "
		}
		out += x
	}
	return out
}

func (r *mutationResolver) ConfigureHealth(ctx context.Context, minFreeGb int) (int, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return 0, err
	}
	if minFreeGb < 1 || minFreeGb > 100000 {
		return 0, errors.New("enter a number of GB from 1 up")
	}
	cfg := config.GetInstance()
	cfg.SetInt(config.HealthMinFreeGB, minFreeGb)
	if err := cfg.Write(); err != nil {
		return 0, err
	}
	return minFreeGb, nil
}

func (r *queryResolver) CleanupSuggestions(ctx context.Context, kind CleanupKind, months *int, limit *int) (*CleanupResult, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	m := 6
	if months != nil {
		m = *months
	}
	n := 100
	if limit != nil && *limit > 0 && *limit <= 500 {
		n = *limit
	}
	k := map[CleanupKind]string{CleanupKindUnwatched: "unwatched", CleanupKindNever: "never", CleanupKindLowRated: "low_rated"}[kind]
	ret := &CleanupResult{Items: []*CleanupItem{}}
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		items, count, total, err := sqlite.CleanupSuggestions(ctx, k, m, n)
		if err != nil {
			return err
		}
		ret.Count, ret.TotalSize = count, total
		ids := make([]int, len(items))
		for i, it := range items {
			ids[i] = it.SceneID
		}
		scenes, err := r.scenesInOrder(ctx, ids)
		if err != nil {
			return err
		}
		byID := map[int]*models.Scene{}
		for _, s := range scenes {
			byID[s.ID] = s
		}
		for _, it := range items {
			s := byID[it.SceneID]
			if s == nil {
				continue
			}
			c := &CleanupItem{Scene: s, Size: it.Size, Plays: it.Plays, Rating: it.Rating}
			if it.LastPlayed != nil {
				t := it.LastPlayed.Timestamp
				c.LastPlayed = &t
			}
			ret.Items = append(ret.Items, c)
		}
		return nil
	})
	return ret, err
}
