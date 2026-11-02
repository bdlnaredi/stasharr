package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
)

func (r *Resolver) Reel() ReelResolver { return &reelResolver{r} }

type reelResolver struct{ *Resolver }

var errReelsNotAllowed = errors.New("your tier does not include reels")

func (r *reelResolver) Scene(ctx context.Context, obj *models.Reel) (*models.Scene, error) {
	return loadSceneByID(ctx, obj.SceneID)
}

func loadSceneByID(ctx context.Context, id int) (*models.Scene, error) {
	s, err := loadersSceneByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, access.ErrHidden
	}
	return s, nil
}

func (r *reelResolver) Cover(ctx context.Context, obj *models.Reel) (string, error) {
	baseURL, _ := ctx.Value(BaseURLCtxKey).(string)
	return fmt.Sprintf("%s/reel/%d/cover.jpg?t=%d", baseURL, obj.ID, obj.CreatedAt.Unix()), nil
}

func (r *reelResolver) Stream(ctx context.Context, obj *models.Reel) (string, error) {
	baseURL, _ := ctx.Value(BaseURLCtxKey).(string)
	return fmt.Sprintf("%s/reel/%d/stream.mp4?t=%d", baseURL, obj.ID, obj.CreatedAt.Unix()), nil
}

func (r *sceneResolver) Reels(ctx context.Context, obj *models.Scene) (ret []*models.Reel, err error) {
	if !access.FromContext(ctx).CanViewReels() {
		return []*models.Reel{}, nil
	}
	err = r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.Reel.FindBySceneID(ctx, obj.ID)
		return err
	})
	if ret == nil {
		ret = []*models.Reel{}
	}
	return ret, err
}

func (r *queryResolver) FindReels(ctx context.Context, page *int, perPage *int, seed *int, startID *string) (*FindReelsResultType, error) {
	if !access.FromContext(ctx).CanViewReels() {
		return nil, errReelsNotAllowed
	}
	p, pp := 1, 40
	if page != nil {
		p = *page
	}
	if perPage != nil {
		pp = *perPage
	}
	if pp > 200 {
		pp = 200
	}
	ret := &FindReelsResultType{}
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		s, first := 0, 0
		if seed != nil {
			s = *seed
		}
		if startID != nil && *startID != "" {
			first, _ = strconv.Atoi(*startID)
		}
		reels, count, err := r.repository.Reel.Query(ctx, p, pp, s, first)
		if err != nil {
			return err
		}
		ret.Reels = reels
		ret.Count = count
		return nil
	})
	if ret.Reels == nil {
		ret.Reels = []*models.Reel{}
	}
	return ret, err
}

func (r *mutationResolver) GenerateReels(ctx context.Context, input GenerateReelsInput) (string, error) {
	if err := access.RequireRole(ctx, access.RoleEditor); err != nil {
		return "", err
	}
	ids, err := stringslice.StringSliceToIntSlice(input.SceneIds)
	if err != nil {
		return "", err
	}
	j := &manager.ReelJob{SceneIDs: ids}
	if input.Duration != nil {
		if *input.Duration < 5 || *input.Duration > 90 {
			return "", errors.New("reel length must be between 5 and 90 seconds")
		}
		j.Duration = *input.Duration
	}
	if input.Count != nil {
		if *input.Count < 1 || *input.Count > 5 {
			return "", errors.New("reels per scene must be between 1 and 5")
		}
		j.Count = *input.Count
	}
	if input.Overwrite != nil {
		j.Overwrite = *input.Overwrite
	}
	// jobs run without the requesting user, so they see the whole library
	jobID := manager.GetInstance().JobManager.Add(context.WithoutCancel(access.WithPrincipal(ctx, nil)), "Making reels...", j)
	return strconv.Itoa(jobID), nil
}

func (r *mutationResolver) ReelDestroy(ctx context.Context, id string) (bool, error) {
	if err := access.RequireRole(ctx, access.RoleEditor); err != nil {
		return false, err
	}
	rid, err := strconv.Atoi(id)
	if err != nil {
		return false, fmt.Errorf("invalid id %q", id)
	}
	var reel *models.Reel
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		reel, err = r.repository.Reel.Find(ctx, rid)
		if err != nil {
			return err
		}
		if reel == nil {
			return fmt.Errorf("reel %d not found", rid)
		}
		return r.repository.Reel.Destroy(ctx, rid)
	}); err != nil {
		return false, err
	}
	_ = os.Remove(reel.Path)
	_ = os.Remove(manager.ReelCoverPath(reel.Path))
	return true, nil
}

func watcherConfig() *LibraryWatcherConfig {
	mgr := manager.GetInstance()
	cfg := mgr.Config
	ret := &LibraryWatcherConfig{
		Enabled:         cfg.GetWatcherEnabled(),
		DebounceSeconds: cfg.GetWatcherDebounceSeconds(),
		Identify:        cfg.GetWatcherIdentify(),
		GenerateReels:   cfg.GetWatcherGenerateReels(),
		GenerateMedia:   cfg.GetWatcherGenerateMedia(),
		CreateMissing:   cfg.GetWatcherCreateMissing(),
		AudioEnabled:    cfg.GetAudioEnabled(),
		AudioExtensions: cfg.GetAudioExtensions(),
		PollMinutes:     cfg.GetWatcherPollMinutes(),
		ReelDuration:    cfg.GetReelDuration(),
		ReelCount:       cfg.GetReelCount(),
		IdentifySources: manager.IdentifySourceNames(cfg),
		WatchedPaths:    []string{},
	}
	if ret.IdentifySources == nil {
		ret.IdentifySources = []string{}
	}
	if mgr.Watcher != nil {
		ret.Running = mgr.Watcher.Running()
		if ret.Running {
			ret.WatchedPaths = mgr.Watcher.WatchedPaths()
		}
		if at, result := mgr.Watcher.Status(); !at.IsZero() {
			ret.LastCheckAt = &at
			ret.LastResult = &result
		}
	}
	return ret
}

func (r *queryResolver) LibraryWatcherConfig(ctx context.Context) (*LibraryWatcherConfig, error) {
	if err := access.RequireRole(ctx, access.RoleEditor); err != nil {
		return nil, err
	}
	return watcherConfig(), nil
}

func (r *mutationResolver) ConfigureLibraryWatcher(ctx context.Context, input LibraryWatcherConfigInput) (*LibraryWatcherConfig, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	mgr := manager.GetInstance()
	cfg := mgr.Config

	if input.Enabled != nil {
		cfg.SetBool(config.WatcherEnabled, *input.Enabled)
	}
	if input.DebounceSeconds != nil {
		if *input.DebounceSeconds < 5 || *input.DebounceSeconds > 3600 {
			return nil, errors.New("wait time must be between 5 and 3600 seconds")
		}
		cfg.SetInt(config.WatcherDebounceSeconds, *input.DebounceSeconds)
	}
	if input.Identify != nil {
		cfg.SetBool(config.WatcherIdentify, *input.Identify)
	}
	if input.GenerateReels != nil {
		cfg.SetBool(config.WatcherGenerateReels, *input.GenerateReels)
	}
	if input.GenerateMedia != nil {
		cfg.SetBool(config.WatcherGenerateMedia, *input.GenerateMedia)
	}
	if input.CreateMissing != nil {
		cfg.SetBool(config.WatcherCreateMissing, *input.CreateMissing)
	}
	if input.AudioEnabled != nil {
		cfg.SetBool(config.AudioEnabled, *input.AudioEnabled)
	}
	if input.AudioExtensions != nil {
		var exts []string
		for _, e := range input.AudioExtensions {
			e = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(e), "."))
			if e == "" {
				continue
			}
			if slices.Contains(cfg.GetVideoExtensions(), e) {
				return nil, fmt.Errorf("%s is already a video extension", e)
			}
			exts = append(exts, e)
		}
		if len(exts) == 0 {
			exts = config.DefaultAudioExtensions
		}
		cfg.SetInterface(config.AudioExtensions, exts)
	}
	if input.PollMinutes != nil {
		if *input.PollMinutes < 0 || *input.PollMinutes > 1440 {
			return nil, errors.New("poll interval must be between 0 and 1440 minutes")
		}
		cfg.SetInt(config.WatcherPollMinutes, *input.PollMinutes)
	}
	if input.ReelDuration != nil {
		if *input.ReelDuration < 5 || *input.ReelDuration > 90 {
			return nil, errors.New("reel length must be between 5 and 90 seconds")
		}
		cfg.SetFloat(config.ReelDuration, *input.ReelDuration)
	}
	if input.ReelCount != nil {
		if *input.ReelCount < 1 || *input.ReelCount > 5 {
			return nil, errors.New("reels per scene must be between 1 and 5")
		}
		cfg.SetInt(config.ReelCount, *input.ReelCount)
	}
	if err := cfg.Write(); err != nil {
		return nil, err
	}

	if mgr.Watcher != nil {
		mgr.Watcher.Restart()
	}
	return watcherConfig(), nil
}

func (r *mutationResolver) LibraryWatcherCheckNow(ctx context.Context) (bool, error) {
	if err := access.RequireRole(ctx, access.RoleEditor); err != nil {
		return false, err
	}
	w := manager.GetInstance().Watcher
	if w == nil {
		return false, errors.New("auto-import is not available")
	}
	if err := w.CheckNow(); err != nil {
		return false, err
	}
	return true, nil
}

func (r *mutationResolver) LibraryWatcherReidentify(ctx context.Context) (string, error) {
	if err := access.RequireRole(ctx, access.RoleEditor); err != nil {
		return "", err
	}
	j := job.MakeJobExec(manager.ReidentifyIncomplete)
	jobID := manager.GetInstance().JobManager.Add(context.WithoutCancel(access.WithPrincipal(ctx, nil)), "Auto-import: identify incomplete scenes", j)
	return strconv.Itoa(jobID), nil
}
