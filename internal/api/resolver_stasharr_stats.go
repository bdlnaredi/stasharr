package api

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sqlite"
)

// sizes of the generated folders take a while to add up, so they are kept
// for a few minutes
var generatedSizes struct {
	sync.Mutex
	at      time.Time
	folders []*StatsFolder
	total   float64
}

const generatedSizesFor = 10 * time.Minute

func generatedFolderSizes(ctx context.Context) ([]*StatsFolder, float64) {
	generatedSizes.Lock()
	defer generatedSizes.Unlock()
	if time.Since(generatedSizes.at) < generatedSizesFor && generatedSizes.folders != nil {
		return generatedSizes.folders, generatedSizes.total
	}
	base := config.GetInstance().GetGeneratedPath()
	folders := []*StatsFolder{}
	total := 0.0
	entries, err := os.ReadDir(base)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			var size int64
			_ = filepath.WalkDir(filepath.Join(base, e.Name()), func(_ string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if !d.IsDir() {
					if info, err := d.Info(); err == nil {
						size += info.Size()
					}
				}
				return nil
			})
			if size > 0 {
				folders = append(folders, &StatsFolder{Name: e.Name(), Bytes: float64(size)})
				total += float64(size)
			}
		}
	}
	sort.Slice(folders, func(i, j int) bool { return folders[i].Bytes > folders[j].Bytes })
	generatedSizes.at, generatedSizes.folders, generatedSizes.total = time.Now(), folders, total
	return folders, total
}

func statsCounts(list []sqlite.StatsCount) []*StatsCount {
	ret := make([]*StatsCount, len(list))
	for i, c := range list {
		ret[i] = &StatsCount{ID: strconv.Itoa(c.ID), Name: c.Name, Count: c.Count}
	}
	return ret
}

func statsTime(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	if t, ok := sqlite.ParseStatsTime(*s); ok {
		return &t
	}
	return nil
}

func (r *queryResolver) StasharrStats(ctx context.Context, days *int) (*StasharrStats, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	d := 30
	if days != nil && *days > 0 && *days <= 3650 {
		d = *days
	}
	since := time.Now().AddDate(0, 0, -d)
	ret := &StasharrStats{
		Days:          d,
		AddedByMonth:  []*StatsMonth{},
		Resolutions:   []*StatsCount{},
		PlaysByDay:    []*StatsDay{},
		TopScenes:     []*StatsScene{},
		TopPerformers: []*StatsCount{},
		TopTags:       []*StatsCount{},
		TopReels:      []*StatsReel{},
		Users:         []*StatsUser{},
		RecentPlays:   []*StatsPlay{},
	}

	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		t, err := sqlite.StatsTotalsQuery(ctx)
		if err != nil {
			return err
		}
		ret.Totals = &StatsTotals{
			Scenes: t.Scenes, Duration: t.Duration, Size: t.Size, Performers: t.Performers,
			Tags: t.Tags, Studios: t.Studios, Reels: t.Reels, ReelViews: t.ReelViews,
			Comments: t.Comments, Users: t.Users + 1, Plays: t.Plays,
		}

		months, err := sqlite.StatsAddedByMonth(ctx, 12)
		if err != nil {
			return err
		}
		for _, m := range months {
			ret.AddedByMonth = append(ret.AddedByMonth, &StatsMonth{Month: m.Month, Scenes: m.Scenes, Size: m.Size})
		}

		res, err := sqlite.StatsResolutions(ctx)
		if err != nil {
			return err
		}
		ret.Resolutions = statsCounts(res)

		days, err := sqlite.StatsPlaysByDay(ctx, since)
		if err != nil {
			return err
		}
		for _, day := range days {
			ret.PlaysByDay = append(ret.PlaysByDay, &StatsDay{Day: day.Day, Plays: day.Plays})
		}

		top, err := sqlite.StatsTopScenes(ctx, since, 10)
		if err != nil {
			return err
		}
		for _, c := range top {
			s, err := r.repository.Scene.Find(ctx, c.ID)
			if err != nil {
				return err
			}
			if s != nil {
				ret.TopScenes = append(ret.TopScenes, &StatsScene{Scene: s, Plays: c.Count})
			}
		}

		perf, err := sqlite.StatsTopPerformers(ctx, since, 10)
		if err != nil {
			return err
		}
		ret.TopPerformers = statsCounts(perf)

		tags, err := sqlite.StatsTopTags(ctx, since, 10)
		if err != nil {
			return err
		}
		ret.TopTags = statsCounts(tags)

		reels, err := sqlite.StatsTopReels(ctx, 10)
		if err != nil {
			return err
		}
		for _, c := range reels {
			sceneID, _ := strconv.Atoi(c.Name)
			s, err := r.repository.Scene.Find(ctx, sceneID)
			if err != nil {
				return err
			}
			if s != nil {
				ret.TopReels = append(ret.TopReels, &StatsReel{ReelID: strconv.Itoa(c.ID), Scene: s, Views: c.Count})
			}
		}

		users, err := sqlite.StatsUsers(ctx, since)
		if err != nil {
			return err
		}
		owner := config.GetInstance().GetUsername()
		if owner == "" {
			owner = "Owner"
		}
		for _, u := range users {
			name := u.Name
			if u.UserID == 0 {
				name = owner
			}
			ret.Users = append(ret.Users, &StatsUser{
				ID: strconv.Itoa(u.UserID), Name: name, Role: u.Role, Plays: u.Plays,
				LastPlay: statsTime(u.LastPlay), LastLogin: statsTime(u.LastLogin),
			})
		}

		plays, err := sqlite.StatsRecentPlays(ctx, 30)
		if err != nil {
			return err
		}
		scenes := map[int]*models.Scene{}
		for _, p := range plays {
			s, ok := scenes[p.SceneID]
			if !ok {
				if s, err = r.repository.Scene.Find(ctx, p.SceneID); err != nil {
					return err
				}
				scenes[p.SceneID] = s
			}
			at, ok := sqlite.ParseStatsTime(p.At)
			if s == nil || !ok {
				continue
			}
			name := p.Name
			if p.UserID == 0 {
				name = owner
			}
			ret.RecentPlays = append(ret.RecentPlays, &StatsPlay{User: name, Scene: s, At: at})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	ret.Generated, ret.GeneratedTotal = generatedFolderSizes(ctx)
	return ret, nil
}
