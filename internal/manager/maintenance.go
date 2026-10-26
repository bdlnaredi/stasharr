package manager

// Scheduled maintenance: nightly database backups that keep the newest few,
// restoring a backup, and refreshing metadata from StashDB and other
// stash-boxes on a schedule.

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stashapp/stash/internal/identify"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/scraper"
	"github.com/stashapp/stash/pkg/sqlite"
)

// AutoBackupDir is the folder, inside the backup folder, that holds the
// scheduled backups. Only files in it are deleted when old.
const AutoBackupDir = "stasharr-auto"

var maintenanceOnce sync.Once

// startMaintenanceLoop checks every few minutes whether a scheduled backup
// or metadata refresh is due.
func (s *Manager) startMaintenanceLoop() {
	maintenanceOnce.Do(func() {
		go func() {
			time.Sleep(3 * time.Minute)
			for {
				s.runDueMaintenance()
				time.Sleep(10 * time.Minute)
			}
		}()
	})
}

var maintenanceMu sync.Mutex

func (s *Manager) runDueMaintenance() {
	if s.Database == nil || s.Database.Ready() != nil || s.Database.NeedsMigration() {
		return
	}
	maintenanceMu.Lock()
	defer maintenanceMu.Unlock()

	cfg := s.Config
	now := time.Now()
	if now.Hour() < cfg.GetMaintenanceHour() {
		return
	}
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if cfg.GetBackupScheduleEnabled() && cfg.GetLastRun(config.BackupLastRun).Before(startOfToday) {
		cfg.SetLastRun(config.BackupLastRun, now)
		_ = cfg.Write()
		s.JobManager.Add(context.Background(), "Scheduled backup", job.MakeJobExec(func(ctx context.Context, progress *job.Progress) error {
			_, err := s.ScheduledBackup()
			return err
		}))
	}

	if cfg.GetResyncEnabled() {
		last := cfg.GetLastRun(config.ResyncLastRun)
		due := last.Add(time.Duration(cfg.GetResyncIntervalDays()) * 24 * time.Hour)
		// run at the maintenance hour, not whenever the interval ends
		if !due.After(now) || last.IsZero() {
			cfg.SetLastRun(config.ResyncLastRun, now)
			_ = cfg.Write()
			s.StartResync(context.Background())
		}
	}
}

// ---------------------------------------------------------------- backups

// BackupFile is a database backup on disk.
type BackupFile struct {
	// ID is "auto/<name>" or "manual/<name>"
	ID            string
	Name          string
	Scheduled     bool
	Size          int64
	CreatedAt     time.Time
	SchemaVersion int
	IncludesBlobs bool
}

func (s *Manager) autoBackupDir() string {
	return filepath.Join(s.Config.GetBackupDirectoryPathOrDefault(), AutoBackupDir)
}

// ScheduledBackup makes a backup in the scheduled backups folder and deletes
// the oldest ones beyond the number to keep.
func (s *Manager) ScheduledBackup() (string, error) {
	cfg := s.Config
	path, name, err := s.BackupDatabase(false, cfg.GetBackupIncludeBlobs())
	if err != nil {
		Notify(EventBackup, "Scheduled backup failed", err.Error())
		return "", err
	}
	dir := s.autoBackupDir()
	if err := fsutil.EnsureDir(dir); err != nil {
		return "", err
	}
	dest := filepath.Join(dir, name)
	if err := os.Rename(path, dest); err != nil {
		// a different disk: copy instead
		if err := fsutil.CopyFile(path, dest); err != nil {
			Notify(EventBackup, "Scheduled backup failed", err.Error())
			return "", fmt.Errorf("moving backup to %s: %w", dir, err)
		}
		_ = os.Remove(path)
	}
	removed := s.pruneBackups()
	var size int64
	if info, err := os.Stat(dest); err == nil {
		size = info.Size()
	}
	logger.Infof("[backup] saved %s", dest)
	msg := fmt.Sprintf("%s (%s)", name, humanSize(size))
	if removed > 0 {
		msg += fmt.Sprintf(". Removed %d old backup(s).", removed)
	}
	Notify(EventBackup, "Backup finished", msg)
	return dest, nil
}

// pruneBackups deletes scheduled backups beyond the number to keep.
func (s *Manager) pruneBackups() int {
	all, err := s.ListBackups()
	if err != nil {
		return 0
	}
	keep := s.Config.GetBackupKeep()
	removed := 0
	n := 0
	for _, b := range all {
		if !b.Scheduled {
			continue
		}
		n++
		if n <= keep {
			continue
		}
		if err := os.Remove(filepath.Join(s.autoBackupDir(), b.Name)); err == nil {
			removed++
		}
	}
	return removed
}

// ListBackups returns the scheduled and manual backups, newest first.
func (s *Manager) ListBackups() ([]BackupFile, error) {
	dbBase := filepath.Base(s.Config.GetDatabasePath())
	var ret []BackupFile
	read := func(dir string, scheduled bool) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			ver, ok := backupSchemaVersion(dbBase, name)
			if !ok {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			prefix := "manual/"
			if scheduled {
				prefix = "auto/"
			}
			ret = append(ret, BackupFile{
				ID:            prefix + name,
				Name:          name,
				Scheduled:     scheduled,
				Size:          info.Size(),
				CreatedAt:     info.ModTime(),
				SchemaVersion: ver,
				IncludesBlobs: strings.HasSuffix(name, ".zip"),
			})
		}
	}
	read(s.autoBackupDir(), true)
	read(s.Config.GetBackupDirectoryPathOrDefault(), false)
	sort.Slice(ret, func(i, j int) bool { return ret[i].CreatedAt.After(ret[j].CreatedAt) })
	return ret, nil
}

// backupSchemaVersion reads the version from a name like
// stash-go.sqlite.89.20261026_040000[.zip].
func backupSchemaVersion(dbBase, name string) (int, bool) {
	if !strings.HasPrefix(name, dbBase+".") {
		return 0, false
	}
	rest := strings.TrimSuffix(strings.TrimPrefix(name, dbBase+"."), ".zip")
	parts := strings.Split(rest, ".")
	if len(parts) != 2 || len(parts[1]) != len("20060102_150405") {
		return 0, false
	}
	v, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, false
	}
	return v, true
}

func (s *Manager) backupPath(id string) (string, error) {
	kind, name, ok := strings.Cut(id, "/")
	if !ok || name == "" || strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		return "", errors.New("unknown backup")
	}
	var dir string
	switch kind {
	case "auto":
		dir = s.autoBackupDir()
	case "manual":
		dir = s.Config.GetBackupDirectoryPathOrDefault()
	default:
		return "", errors.New("unknown backup")
	}
	if _, ok := backupSchemaVersion(filepath.Base(s.Config.GetDatabasePath()), name); !ok {
		return "", errors.New("unknown backup")
	}
	p := filepath.Join(dir, name)
	if _, err := os.Stat(p); err != nil {
		return "", errors.New("the backup file is missing")
	}
	return p, nil
}

// DeleteBackup removes a backup file.
func (s *Manager) DeleteBackup(id string) error {
	p, err := s.backupPath(id)
	if err != nil {
		return err
	}
	return os.Remove(p)
}

// RestoreBackup replaces the database with a backup. A backup of the
// current database is made first, so a restore can be undone.
func (s *Manager) RestoreBackup(ctx context.Context, id string) error {
	src, err := s.backupPath(id)
	if err != nil {
		return err
	}
	dbBase := filepath.Base(s.Config.GetDatabasePath())
	ver, _ := backupSchemaVersion(dbBase, filepath.Base(src))
	if uint(ver) > s.Database.AppSchemaVersion() {
		return fmt.Errorf("this backup is from a newer version of Stasharr (database version %d)", ver)
	}

	maintenanceMu.Lock()
	defer maintenanceMu.Unlock()

	// a safety copy of the current database
	if _, _, err := s.BackupDatabase(false, false); err != nil {
		return fmt.Errorf("could not back up the current database first: %w", err)
	}

	tmpDir := s.Paths.Generated.Tmp
	if err := fsutil.EnsureDir(tmpDir); err != nil {
		return err
	}
	staged := filepath.Join(tmpDir, "restore.sqlite")
	_ = os.Remove(staged)
	isZip := strings.HasSuffix(src, ".zip")
	if isZip {
		if err := extractZipEntry(src, dbBase, staged); err != nil {
			return err
		}
	} else if err := fsutil.CopyFile(src, staged); err != nil {
		return fmt.Errorf("copying backup: %w", err)
	}

	s.Watcher.Stop()
	s.JobManager.CancelAll()
	time.Sleep(2 * time.Second)

	dbPath := s.Database.DatabasePath()
	if err := s.Database.Close(); err != nil {
		return fmt.Errorf("closing database: %w", err)
	}
	for _, f := range []string{dbPath + "-wal", dbPath + "-shm"} {
		_ = os.Remove(f)
	}
	if err := os.Rename(staged, dbPath); err != nil {
		if err := fsutil.CopyFile(staged, dbPath); err != nil {
			// put the database back as it was
			_ = s.Database.Open(dbPath)
			return fmt.Errorf("replacing database: %w", err)
		}
		_ = os.Remove(staged)
	}

	if isZip && s.Config.GetBlobsStorage() == config.BlobStorageTypeFilesystem {
		if err := extractZipDir(src, "blobs/", s.Config.GetBlobsPath()); err != nil {
			logger.Warnf("[backup] restoring images: %v", err)
		}
	}

	if err := s.Database.Open(dbPath); err != nil {
		var mErr *sqlite.MigrationNeededError
		if errors.As(err, &mErr) {
			if err := s.Database.RunAllMigrations(); err != nil {
				return fmt.Errorf("updating the restored database: %w", err)
			}
			if err := s.Database.Open(dbPath); err != nil {
				return fmt.Errorf("opening the restored database: %w", err)
			}
		} else {
			return fmt.Errorf("opening the restored database: %w", err)
		}
	}
	logger.Infof("[backup] restored %s", filepath.Base(src))
	go s.Watcher.Restart()
	return nil
}

func extractZipEntry(zipPath, name, dest string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		return writeZipFile(f, dest)
	}
	return fmt.Errorf("the backup has no %s", name)
}

func extractZipDir(zipPath, prefix, destDir string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()
	cleanDest := filepath.Clean(destDir)
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, prefix) || strings.HasSuffix(f.Name, "/") {
			continue
		}
		dest := filepath.Join(cleanDest, filepath.FromSlash(strings.TrimPrefix(f.Name, prefix)))
		if !strings.HasPrefix(dest, cleanDest+string(filepath.Separator)) {
			continue
		}
		if err := fsutil.EnsureDir(filepath.Dir(dest)); err != nil {
			return err
		}
		if err := writeZipFile(f, dest); err != nil {
			return err
		}
	}
	return nil
}

func writeZipFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, rc); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func humanSize(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	default:
		return fmt.Sprintf("%d KB", n/1024)
	}
}

// ---------------------------------------------------------------- metadata refresh

// StartResync queues a metadata refresh from every stash-box for the scenes,
// performers and studios already matched to it. Returns the job id.
func (s *Manager) StartResync(ctx context.Context) int {
	return s.JobManager.Add(ctx, "Refreshing metadata from stash-boxes", job.MakeJobExec(func(ctx context.Context, progress *job.Progress) error {
		return s.resync(ctx, progress)
	}))
}

func (s *Manager) resync(ctx context.Context, progress *job.Progress) error {
	cfg := s.Config
	boxes := cfg.GetStashBoxes()
	if len(boxes) == 0 {
		return errors.New("no stash-boxes: add StashDB under Settings > Metadata Providers")
	}

	var summary []string
	for _, box := range boxes {
		if job.IsCancelled(ctx) {
			return nil
		}
		label := box.Name
		if label == "" {
			label = box.Endpoint
		}
		var tasks []Task
		perf, stud := 0, 0
		input := StashBoxBatchTagInput{Refresh: true, CreateParent: true}
		if cfg.GetResyncPerformers() {
			t, err := s.batchTagAllPerformers(ctx, input, box)
			if err != nil {
				return err
			}
			perf = len(t)
			tasks = append(tasks, t...)
		}
		if cfg.GetResyncStudios() {
			t, err := s.batchTagAllStudios(ctx, input, box)
			if err != nil {
				return err
			}
			stud = len(t)
			tasks = append(tasks, t...)
		}

		sceneIDs, err := s.scenesMatchedTo(ctx, box.Endpoint)
		if err != nil {
			return err
		}
		if !cfg.GetResyncScenes() {
			sceneIDs = nil
		}

		logger.Infof("[resync] %s: %d performers, %d studios, %d scenes", label, perf, stud, len(sceneIDs))
		progress.SetTotal(len(tasks) + 1)
		for _, task := range tasks {
			if job.IsCancelled(ctx) {
				return nil
			}
			progress.ExecuteTask(task.GetDescription(), func() { task.Start(ctx) })
			progress.Increment()
		}

		if len(sceneIDs) > 0 {
			endpoint := box.Endpoint
			opts := identify.Options{
				Sources: []*identify.Source{{Source: &scraper.Source{StashBoxEndpoint: &endpoint}}},
			}
			if base := baseIdentifyOptions(cfg); base.Options != nil {
				cp := *base.Options
				opts.Options = &cp
			}
			withCreateMissing(&opts)
			opts.SceneIDs = sceneIDs
			if err := CreateIdentifyJob(opts).Execute(ctx, progress); err != nil {
				logger.Errorf("[resync] scenes: %v", err)
			}
		}
		progress.Increment()
		summary = append(summary, fmt.Sprintf("%s: %d scenes, %d performers, %d studios", label, len(sceneIDs), perf, stud))
	}
	Notify(EventResync, "Metadata refresh finished", strings.Join(summary, "\n"))
	return nil
}

// scenesMatchedTo returns the scenes that have a stash id from the endpoint.
func (s *Manager) scenesMatchedTo(ctx context.Context, endpoint string) ([]string, error) {
	r := s.Repository
	ep := endpoint
	perPage := -1
	var ids []string
	err := r.WithReadTxn(ctx, func(ctx context.Context) error {
		res, err := r.Scene.Query(ctx, models.SceneQueryOptions{
			QueryOptions: models.QueryOptions{FindFilter: &models.FindFilterType{PerPage: &perPage}},
			SceneFilter: &models.SceneFilterType{
				StashIDEndpoint: &models.StashIDCriterionInput{Endpoint: &ep, Modifier: models.CriterionModifierNotNull},
			},
		})
		if err != nil {
			return err
		}
		for _, id := range res.IDs {
			ids = append(ids, strconv.Itoa(id))
		}
		return nil
	})
	return ids, err
}
