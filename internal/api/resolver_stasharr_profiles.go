package api

// Stasharr: profiles and the monthly recap.

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sqlite"
)

func profileSignedIn(ctx context.Context) (*access.Principal, error) {
	p := access.FromContext(ctx)
	if p == nil {
		return nil, errors.New("profiles need sign in to be turned on in Settings > Security")
	}
	return p, nil
}

func profileID(id string) (int, error) {
	n, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("invalid id %q", id)
	}
	return n, nil
}

func (r *queryResolver) SiteProfiles(ctx context.Context) ([]*SiteProfile, error) {
	p := access.FromContext(ctx)
	if p == nil {
		return []*SiteProfile{}, nil
	}
	list, err := manager.Profiles(ctx, p)
	if err != nil {
		return nil, err
	}
	ret := make([]*SiteProfile, len(list))
	for i, x := range list {
		ret[i] = &SiteProfile{
			ID: strconv.Itoa(x.ID), Name: x.Name, Color: x.Color,
			HasPin: x.HasPin, IsMain: x.Main, IsCurrent: x.Current,
		}
	}
	return ret, nil
}

func (r *mutationResolver) SiteProfileCreate(ctx context.Context, name string, color *string) (*SiteProfile, error) {
	p, err := profileSignedIn(ctx)
	if err != nil {
		return nil, err
	}
	c := ""
	if color != nil {
		c = *color
	}
	x, err := manager.CreateProfile(ctx, p, name, c)
	if err != nil {
		return nil, err
	}
	return &SiteProfile{ID: strconv.Itoa(x.ID), Name: x.Name, Color: x.Color}, nil
}

func (r *mutationResolver) SiteProfileUpdate(ctx context.Context, id string, name string, color *string) (bool, error) {
	p, err := profileSignedIn(ctx)
	if err != nil {
		return false, err
	}
	n, err := profileID(id)
	if err != nil {
		return false, err
	}
	c := ""
	if color != nil {
		c = *color
	}
	if err := manager.UpdateProfile(ctx, p, n, name, c); err != nil {
		return false, err
	}
	return true, nil
}

func (r *mutationResolver) SiteProfileDelete(ctx context.Context, id string) (bool, error) {
	p, err := profileSignedIn(ctx)
	if err != nil {
		return false, err
	}
	n, err := profileID(id)
	if err != nil {
		return false, err
	}
	if err := manager.DeleteProfile(ctx, p, n); err != nil {
		return false, err
	}
	return true, nil
}

func (r *mutationResolver) SiteProfilePin(ctx context.Context, id string, pin string) (bool, error) {
	p, err := profileSignedIn(ctx)
	if err != nil {
		return false, err
	}
	n, err := profileID(id)
	if err != nil {
		return false, err
	}
	if err := manager.SetProfilePin(ctx, p, n, pin); err != nil {
		return false, err
	}
	return true, nil
}

// ---------------------------------------------------------------- recap

func (r *queryResolver) RecapMonths(ctx context.Context) (ret []string, err error) {
	err = r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = sqlite.RecapMonths(ctx, sqlite.PlaylistUserID(ctx), 24)
		return err
	})
	if ret == nil {
		ret = []string{}
	}
	return ret, err
}

func (r *queryResolver) MonthRecap(ctx context.Context, month *string) (*MonthRecap, error) {
	var ret *MonthRecap
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		uid := sqlite.PlaylistUserID(ctx)
		m := ""
		if month != nil {
			if _, err := time.Parse("2006-01", *month); err != nil {
				return fmt.Errorf("month should look like 2026-11")
			}
			m = *month
		} else {
			months, err := sqlite.RecapMonths(ctx, uid, 1)
			if err != nil {
				return err
			}
			if len(months) == 0 {
				return nil
			}
			m = months[0]
		}
		rc, err := sqlite.MonthRecap(ctx, uid, m)
		if err != nil {
			return err
		}
		ret = &MonthRecap{
			Month: rc.Month, Plays: rc.Plays, Scenes: rc.Scenes, Seconds: rc.Seconds,
			ActiveDays: rc.ActiveDays, LongestRun: rc.LongestRun, Likes: rc.Likes,
			TopScenePlays: rc.TopScenePlay,
			Hours:         rc.Hours[:],
			Weekdays:      rc.Weekdays[:],
			Models:        []*RecapPerformer{},
			Categories:    []*RecapTag{},
			Studios:       []*RecapStudio{},
		}
		for _, c := range rc.Models {
			x, err := r.repository.Performer.Find(ctx, c.ID)
			if err != nil {
				return err
			}
			if x != nil {
				ret.Models = append(ret.Models, &RecapPerformer{Performer: x, Plays: c.Plays})
			}
		}
		for _, c := range rc.Tags {
			x, err := r.repository.Tag.Find(ctx, c.ID)
			if err != nil {
				return err
			}
			if x != nil {
				ret.Categories = append(ret.Categories, &RecapTag{Tag: x, Plays: c.Plays})
			}
		}
		for _, c := range rc.Studios {
			x, err := r.repository.Studio.Find(ctx, c.ID)
			if err != nil {
				return err
			}
			if x != nil {
				ret.Studios = append(ret.Studios, &RecapStudio{Studio: x, Plays: c.Plays})
			}
		}
		if rc.TopSceneID != 0 {
			var s *models.Scene
			s, err = r.repository.Scene.Find(ctx, rc.TopSceneID)
			if err != nil {
				return err
			}
			ret.TopScene = s
		}
		return nil
	})
	return ret, err
}
