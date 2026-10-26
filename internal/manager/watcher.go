package manager

// The library watcher notices new files in the library folders and runs
// scan -> identify -> generate -> reels for just those files.
//
// File system events are used where the OS provides them. Network shares
// (SMB/NFS) and some Docker setups do not deliver events, so the watcher
// also polls for changes on an interval.

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/stashapp/stash/internal/identify"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/scraper"
	"github.com/stashapp/stash/pkg/sqlite"
)

// maximum number of folders to register for events before relying on polling
const maxWatchedDirs = 20000

type LibraryWatcher struct {
	mu       sync.Mutex
	running  bool
	cancel   context.CancelFunc
	watcher  *fsnotify.Watcher
	pending  map[string]struct{}
	timer    *time.Timer
	roots    []string
	lastPoll time.Time
	busy     bool
	busyAt   time.Time

	lastCheck  time.Time
	lastResult string
}

// Status returns when the watcher last checked for new files and what it found.
func (w *LibraryWatcher) Status() (time.Time, string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lastCheck, w.lastResult
}

func (w *LibraryWatcher) setResult(msg string) {
	w.mu.Lock()
	w.lastCheck = time.Now()
	w.lastResult = msg
	w.mu.Unlock()
}

// CheckNow scans every library folder for new files straight away.
func (w *LibraryWatcher) CheckNow() error {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return errors.New("auto-import is switched off")
	}
	w.clearStaleBusy()
	if w.busy {
		w.mu.Unlock()
		return errors.New("a check is already running")
	}
	roots := slices.Clone(w.roots)
	w.busy = true
	w.busyAt = time.Now()
	w.lastPoll = time.Now()
	w.mu.Unlock()

	logger.Info("[watcher] checking library for new files")
	w.process(roots)
	return nil
}

var errWatcherBusy = errors.New("a check is already running")

// ImportNow scans, identifies and prepares the given folders straight away,
// even when the watcher is switched off. Used for files restored from the
// trash bin.
func (w *LibraryWatcher) ImportNow(dirs []string) error {
	w.mu.Lock()
	w.clearStaleBusy()
	if w.busy {
		w.mu.Unlock()
		return errWatcherBusy
	}
	w.busy = true
	w.busyAt = time.Now()
	w.mu.Unlock()

	w.process(collapsePaths(dirs))
	return nil
}

func NewLibraryWatcher() *LibraryWatcher {
	return &LibraryWatcher{pending: map[string]struct{}{}}
}

// Running reports whether the watcher is active.
func (w *LibraryWatcher) Running() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

// WatchedPaths returns the library folders being watched.
func (w *LibraryWatcher) WatchedPaths() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.roots)
}

// Restart applies the current configuration.
func (w *LibraryWatcher) Restart() {
	w.Stop()
	if config.GetInstance().GetWatcherEnabled() {
		if err := w.Start(); err != nil {
			logger.Errorf("[watcher] could not start: %v", err)
		}
	}
}

// RestartWhenReady starts the watcher once the database is open and
// migrated, for example after the migration page has been completed.
func (w *LibraryWatcher) RestartWhenReady(db *sqlite.Database) {
	for i := 0; i < 24*60; i++ {
		if db.Ready() == nil && !db.NeedsMigration() {
			w.Restart()
			return
		}
		time.Sleep(time.Minute)
	}
}

func (w *LibraryWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.running {
		return
	}
	w.cancel()
	if w.watcher != nil {
		_ = w.watcher.Close()
		w.watcher = nil
	}
	if w.timer != nil {
		w.timer.Stop()
	}
	w.running = false
	logger.Info("[watcher] stopped")
}

func (w *LibraryWatcher) Start() error {
	cfg := config.GetInstance()
	var roots []string
	for _, s := range cfg.GetStashPaths() {
		if s.ExcludeVideo {
			continue
		}
		roots = append(roots, s.Path)
	}
	if len(roots) == 0 {
		return errors.New("no library folders configured")
	}

	ctx, cancel := context.WithCancel(context.Background())

	w.mu.Lock()
	w.running = true
	w.cancel = cancel
	w.roots = roots
	w.lastPoll = time.Now()
	w.mu.Unlock()

	fw, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Warnf("[watcher] file events unavailable, polling only: %v", err)
	} else {
		w.mu.Lock()
		w.watcher = fw
		w.mu.Unlock()
		go w.addTree(fw, roots)
		go w.eventLoop(ctx, fw)
	}

	go w.pollLoop(ctx)

	logger.Infof("[watcher] watching %s", strings.Join(roots, ", "))
	return nil
}

func (w *LibraryWatcher) addTree(fw *fsnotify.Watcher, roots []string) {
	count := 0
	for _, root := range roots {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() {
				return nil
			}
			if strings.HasPrefix(d.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			if count >= maxWatchedDirs {
				return filepath.SkipAll
			}
			if err := fw.Add(path); err != nil {
				logger.Debugf("[watcher] cannot watch %s: %v", path, err)
				return nil
			}
			count++
			return nil
		})
	}
	if count >= maxWatchedDirs {
		logger.Warnf("[watcher] library has more than %d folders: the rest is covered by polling", maxWatchedDirs)
	}
	logger.Debugf("[watcher] registered %d folders", count)
}

func (w *LibraryWatcher) eventLoop(ctx context.Context, fw *fsnotify.Watcher) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-fw.Events:
			if !ok {
				return
			}
			w.handleEvent(fw, ev)
		case err, ok := <-fw.Errors:
			if !ok {
				return
			}
			logger.Warnf("[watcher] %v", err)
		}
	}
}

func (w *LibraryWatcher) handleEvent(fw *fsnotify.Watcher, ev fsnotify.Event) {
	if !ev.Has(fsnotify.Create) && !ev.Has(fsnotify.Write) && !ev.Has(fsnotify.Rename) {
		return
	}
	base := filepath.Base(ev.Name)
	if strings.HasPrefix(base, ".") || strings.HasSuffix(base, ".part") || strings.HasSuffix(base, ".!qB") || strings.HasSuffix(base, ".tmp") {
		return
	}

	info, err := os.Stat(ev.Name)
	if err != nil {
		return
	}
	if info.IsDir() {
		// a new folder: watch it and scan its contents
		go w.addTree(fw, []string{ev.Name})
		w.queue(ev.Name)
		return
	}
	if isVideo(ev.Name) {
		w.queue(filepath.Dir(ev.Name))
	}
}

// queue adds a folder to be scanned after the debounce period.
func (w *LibraryWatcher) queue(dir string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.running {
		return
	}
	w.pending[dir] = struct{}{}
	delay := time.Duration(config.GetInstance().GetWatcherDebounceSeconds()) * time.Second
	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(delay, w.flush)
}

func (w *LibraryWatcher) flush() {
	w.mu.Lock()
	if len(w.pending) == 0 {
		w.mu.Unlock()
		return
	}
	w.clearStaleBusy()
	if w.busy {
		// a run is in progress: try again later
		w.timer = time.AfterFunc(30*time.Second, w.flush)
		w.mu.Unlock()
		return
	}
	dirs := make([]string, 0, len(w.pending))
	for d := range w.pending {
		dirs = append(dirs, d)
	}
	w.pending = map[string]struct{}{}
	w.busy = true
	w.busyAt = time.Now()
	w.mu.Unlock()

	dirs = collapsePaths(dirs)
	logger.Infof("[watcher] new files in %s", strings.Join(dirs, ", "))
	w.process(dirs)
}

func (w *LibraryWatcher) pollLoop(ctx context.Context) {
	// first check soon after start, to pick up anything added while stopped
	wait := 2 * time.Minute
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}

		minutes := config.GetInstance().GetWatcherPollMinutes()
		if minutes <= 0 {
			// polling is off: re-check the setting hourly
			wait = time.Hour
			continue
		}
		wait = time.Duration(minutes) * time.Minute

		w.mu.Lock()
		w.clearStaleBusy()
		if w.busy {
			w.mu.Unlock()
			continue
		}
		w.lastPoll = time.Now()
		roots := slices.Clone(w.roots)
		w.busy = true
		w.busyAt = time.Now()
		w.mu.Unlock()

		logger.Debug("[watcher] polling library for changes")
		// a full scan only reads files it has not seen before; moved or
		// copied downloads keep their old modification time, so no time
		// filter is used here
		w.process(roots)
	}
}

// clearStaleBusy recovers if a follow-up job was cancelled before it ran.
// Must be called with the lock held.
func (w *LibraryWatcher) clearStaleBusy() {
	if w.busy && time.Since(w.busyAt) > 6*time.Hour {
		w.busy = false
	}
}

// collapsePaths removes paths that are inside another path in the list.
func collapsePaths(paths []string) []string {
	sort.Strings(paths)
	var ret []string
	for _, p := range paths {
		if len(ret) > 0 {
			last := ret[len(ret)-1]
			if p == last || strings.HasPrefix(p, last+string(filepath.Separator)) {
				continue
			}
		}
		ret = append(ret, p)
	}
	return ret
}

// process queues the scan, then a follow-up job that handles the scenes the
// scan created. Jobs run one at a time, so the follow-up runs after the scan.
func (w *LibraryWatcher) process(dirs []string) {
	mgr := GetInstance()
	ctx := context.Background()

	done := func() {
		w.mu.Lock()
		w.busy = false
		w.mu.Unlock()
	}

	maxID, err := maxSceneID(ctx)
	if err != nil {
		logger.Errorf("[watcher] %v", err)
		w.setResult("Error: " + err.Error())
		done()
		return
	}

	input := ScanMetadataInput{Paths: dirs}
	if def := mgr.Config.GetDefaultScanSettings(); def != nil {
		input.ScanMetadataOptions = *def
	} else {
		input.ScanGenerateCovers = true
		input.ScanGeneratePhashes = true
	}
	// fingerprints are needed to match scenes on ThePornDB and StashDB
	input.ScanGeneratePhashes = true
	input.Rescan = false

	if _, err := mgr.Scan(ctx, input); err != nil {
		logger.Errorf("[watcher] scan: %v", err)
		w.setResult("Scan failed: " + err.Error())
		done()
		return
	}

	follow := job.MakeJobExec(func(ctx context.Context, progress *job.Progress) error {
		defer done()
		return w.afterScan(ctx, progress, maxID)
	})
	mgr.JobManager.Add(ctx, "Auto-import: identify new scenes", follow)
}

func maxSceneID(ctx context.Context) (int, error) {
	r := GetInstance().Repository
	sort := "id"
	dir := models.SortDirectionEnumDesc
	perPage := 1
	var ids []int
	err := r.WithReadTxn(ctx, func(ctx context.Context) error {
		res, err := r.Scene.Query(ctx, models.SceneQueryOptions{
			QueryOptions: models.QueryOptions{
				FindFilter: &models.FindFilterType{Sort: &sort, Direction: &dir, PerPage: &perPage},
			},
		})
		if err != nil {
			return err
		}
		ids = res.IDs
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("finding latest scene: %w", err)
	}
	if len(ids) == 0 {
		return 0, nil
	}
	return ids[0], nil
}

func (w *LibraryWatcher) afterScan(ctx context.Context, progress *job.Progress, sinceID int) error {
	mgr := GetInstance()
	cfg := mgr.Config
	r := mgr.Repository

	newID, err := maxSceneID(ctx)
	if err != nil {
		return err
	}
	if newID <= sinceID {
		logger.Debug("[watcher] no new scenes")
		w.setResult("No new files")
		return nil
	}

	var ids []string
	var intIDs []int
	for id := sinceID + 1; id <= newID; id++ {
		var s *models.Scene
		if err := r.WithReadTxn(ctx, func(ctx context.Context) error {
			var err error
			s, err = r.Scene.Find(ctx, id)
			return err
		}); err != nil {
			return err
		}
		if s == nil {
			continue
		}
		ids = append(ids, strconv.Itoa(id))
		intIDs = append(intIDs, id)
	}
	if len(ids) == 0 {
		return nil
	}
	logger.Infof("[watcher] %d new scene(s)", len(ids))
	w.setResult(fmt.Sprintf("Added %d new scene(s)", len(ids)))

	if cfg.GetWatcherIdentify() {
		opts := identifyOptionsFor(cfg)
		if len(opts.Sources) == 0 {
			logger.Warn("[watcher] no identify sources: add ThePornDB or StashDB under Settings > Metadata Providers")
		} else {
			opts.SceneIDs = ids
			identifyJob := CreateIdentifyJob(opts)
			if err := identifyJob.Execute(ctx, progress); err != nil {
				logger.Errorf("[watcher] identify: %v", err)
			}
		}
	}

	notifyNewScenes(ctx, intIDs, cfg.GetWatcherIdentify())

	if cfg.GetRenamerEnabled() {
		// jobs run in order, so the files are renamed before media is generated
		mgr.JobManager.Add(ctx, "Auto-import: renaming files", &RenameJob{SceneIDs: intIDs})
	}

	if cfg.GetWatcherGenerateMedia() {
		gen := GenerateMetadataInput{
			Covers:   true,
			Previews: true,
			Sprites:  true,
			Phashes:  true,
			SceneIDs: ids,
		}
		if def := cfg.GetDefaultGenerateSettings(); def != nil {
			gen.Previews = def.Previews
			gen.Sprites = def.Sprites
			gen.Markers = def.Markers
			gen.Transcodes = def.Transcodes
			gen.InteractiveHeatmapsSpeeds = def.InteractiveHeatmapsSpeeds
		}
		if _, err := mgr.Generate(ctx, gen); err != nil {
			logger.Errorf("[watcher] generate: %v", err)
		}
	}

	if cfg.GetWatcherGenerateReels() {
		mgr.JobManager.Add(ctx, "Auto-import: making reels", &ReelJob{SceneIDs: intIDs})
	}

	return nil
}

// identifyOptionsFor returns the saved Identify task settings, or a default
// using ThePornDB, then StashDB, then other stash-boxes, then scrapers that
// can identify scenes from their file details.
func identifyOptionsFor(cfg *config.Config) identify.Options {
	ret := baseIdentifyOptions(cfg)
	if cfg.GetWatcherCreateMissing() {
		withCreateMissing(&ret)
	}
	return ret
}

// createMissingFields are the fields whose objects identify can create.
var createMissingFields = []string{"studio", "performers", "tags"}

// withCreateMissing makes identify add studios, performers and tags that are
// not in the library yet, instead of leaving them out. Upstream Stash only
// links existing ones unless "create missing" is ticked for each field.
func withCreateMissing(opts *identify.Options) {
	yes := true
	set := func(m *identify.MetadataOptions, addMissing bool) {
		have := map[string]bool{}
		fields := make([]*identify.FieldOptions, 0, len(m.FieldOptions)+len(createMissingFields))
		for _, f := range m.FieldOptions {
			if f == nil {
				continue
			}
			cp := *f
			if slices.Contains(createMissingFields, cp.Field) && cp.Strategy != identify.FieldStrategyIgnore {
				cp.CreateMissing = &yes
			}
			have[cp.Field] = true
			fields = append(fields, &cp)
		}
		if addMissing {
			for _, name := range createMissingFields {
				if !have[name] {
					fields = append(fields, &identify.FieldOptions{Field: name, Strategy: identify.FieldStrategyMerge, CreateMissing: &yes})
				}
			}
		}
		m.FieldOptions = fields
	}

	if opts.Options == nil {
		opts.Options = &identify.MetadataOptions{}
	} else {
		cp := *opts.Options
		opts.Options = &cp
	}
	set(opts.Options, true)
	// performers with one name, such as many ThePornDB performers, are
	// skipped by default; create them too unless the saved settings say not to
	if opts.Options.SkipSingleNamePerformers == nil {
		no := false
		opts.Options.SkipSingleNamePerformers = &no
	}

	sources := make([]*identify.Source, 0, len(opts.Sources))
	for _, s := range opts.Sources {
		if s == nil {
			continue
		}
		cp := *s
		if cp.Options != nil {
			o := *cp.Options
			// a source's own field settings win over the defaults, so turn
			// create missing on there too
			set(&o, false)
			cp.Options = &o
		}
		sources = append(sources, &cp)
	}
	opts.Sources = sources
}

func baseIdentifyOptions(cfg *config.Config) identify.Options {
	if def := cfg.GetDefaultIdentifySettings(); def != nil && len(def.Sources) > 0 {
		ret := *def
		ret.Paths = nil
		return ret
	}

	var ret identify.Options
	boxes := cfg.GetStashBoxes()
	order := make([]int, len(boxes))
	for i := range boxes {
		order[i] = i
	}
	rank := func(b *models.StashBox) int {
		e := strings.ToLower(b.Endpoint + " " + b.Name)
		switch {
		case strings.Contains(e, "theporndb") || strings.Contains(e, "tpdb"):
			return 0
		case strings.Contains(e, "stashdb"):
			return 1
		default:
			return 2
		}
	}
	sort.SliceStable(order, func(i, j int) bool { return rank(boxes[order[i]]) < rank(boxes[order[j]]) })
	for _, i := range order {
		endpoint := boxes[i].Endpoint
		ret.Sources = append(ret.Sources, &identify.Source{Source: &scraper.Source{StashBoxEndpoint: &endpoint}})
	}

	if cache := GetInstance().ScraperCache; cache != nil {
		for _, s := range cache.ListScrapers([]scraper.ScrapeContentType{scraper.ScrapeContentTypeScene}) {
			if s.Scene == nil || !slices.Contains(s.Scene.SupportedScrapes, scraper.ScrapeTypeFragment) {
				continue
			}
			id := s.ID
			ret.Sources = append(ret.Sources, &identify.Source{Source: &scraper.Source{ScraperID: &id}})
		}
	}

	setOrganized := false
	ret.Options = &identify.MetadataOptions{SetOrganized: &setOrganized}
	return ret
}

// IdentifySourceNames describes the identify sources the watcher uses.
func IdentifySourceNames(cfg *config.Config) []string {
	opts := identifyOptionsFor(cfg)
	boxes := cfg.GetStashBoxes()
	var ret []string
	for _, s := range opts.Sources {
		if s.Source == nil {
			continue
		}
		switch {
		case s.Source.ScraperID != nil:
			name := *s.Source.ScraperID
			if sc := GetInstance().ScraperCache.GetScraper(name); sc != nil {
				name = sc.Name
			}
			ret = append(ret, "Scraper: "+name)
		default:
			sb, err := resolveStashBox(boxes, *s.Source)
			if err == nil && sb != nil {
				label := sb.Name
				if label == "" {
					label = sb.Endpoint
				}
				ret = append(ret, "Stash-box: "+label)
			}
		}
	}
	return ret
}

// ReidentifyIncomplete identifies again every scene that has no studio or no
// performers, using the auto-import identify settings.
func ReidentifyIncomplete(ctx context.Context, progress *job.Progress) error {
	mgr := GetInstance()
	r := mgr.Repository
	var ids []string
	if err := r.WithReadTxn(ctx, func(ctx context.Context) error {
		all, err := r.Scene.All(ctx)
		if err != nil {
			return err
		}
		for _, s := range all {
			if s.StudioID == nil {
				ids = append(ids, strconv.Itoa(s.ID))
				continue
			}
			if err := s.LoadPerformerIDs(ctx, r.Scene); err != nil {
				return err
			}
			if len(s.PerformerIDs.List()) == 0 {
				ids = append(ids, strconv.Itoa(s.ID))
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if len(ids) == 0 {
		logger.Info("[watcher] every scene already has a studio and performers")
		return nil
	}
	opts := identifyOptionsFor(mgr.Config)
	if len(opts.Sources) == 0 {
		return errors.New("no identify sources: add ThePornDB or StashDB under Settings > Metadata Providers")
	}
	logger.Infof("[watcher] identifying %d scene(s) with no studio or performers", len(ids))
	opts.SceneIDs = ids
	return CreateIdentifyJob(opts).Execute(ctx, progress)
}

// notifyNewScenes tells the notification targets about new scenes, and about
// the ones identify could not match.
func notifyNewScenes(ctx context.Context, ids []int, identified bool) {
	r := GetInstance().Repository
	var all, unmatched []string
	_ = r.WithReadTxn(ctx, func(ctx context.Context) error {
		for _, id := range ids {
			s, err := r.Scene.Find(ctx, id)
			if err != nil || s == nil {
				continue
			}
			if err := s.LoadFiles(ctx, r.Scene); err != nil {
				continue
			}
			name := s.DisplayName()
			all = append(all, name)
			if !identified {
				continue
			}
			if err := s.LoadStashIDs(ctx, r.Scene); err != nil {
				continue
			}
			if s.StudioID == nil && len(s.StashIDs.List()) == 0 {
				unmatched = append(unmatched, name)
			}
		}
		return nil
	})
	if len(all) > 0 {
		Notify(EventNewScenes, fmt.Sprintf("%d new scene(s) added", len(all)), listTitles(all, 15))
	}
	if len(unmatched) > 0 {
		Notify(EventUnmatched, fmt.Sprintf("%d new scene(s) could not be identified", len(unmatched)), listTitles(unmatched, 15))
	}
}
