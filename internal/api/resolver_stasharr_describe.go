package api

import (
	"context"
	"fmt"
	"strconv"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stashapp/stash/pkg/sqlite"
)

func (r *queryResolver) SearchMoments(ctx context.Context, query string, limit *int) ([]*SearchMoment, error) {
	n := 24
	if limit != nil && *limit > 0 && *limit <= 100 {
		n = *limit
	}
	// extra hits, as some scenes may be hidden from this viewer
	hits, err := manager.SearchFrames(ctx, query, n*3)
	if err != nil {
		return nil, err
	}
	baseURL, _ := ctx.Value(BaseURLCtxKey).(string)
	ret := []*SearchMoment{}
	err = r.withReadTxn(ctx, func(ctx context.Context) error {
		for _, h := range hits {
			if len(ret) >= n {
				break
			}
			// scene lookups follow the viewer's tier
			s, err := r.repository.Scene.Find(ctx, h.SceneID)
			if err != nil {
				return err
			}
			if s == nil {
				continue
			}
			ret = append(ret, &SearchMoment{
				Scene:   s,
				Seconds: h.Seconds,
				Score:   h.Score,
				Frame:   fmt.Sprintf("%s/scene/%d/frame/%d", baseURL, h.SceneID, int(h.Seconds)),
			})
		}
		return nil
	})
	return ret, err
}

func (r *queryResolver) SearchIndexStatus(ctx context.Context) (*SearchIndexStatus, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	model := manager.SearchModel(ctx)
	ret := &SearchIndexStatus{Ready: model != ""}
	if model != "" {
		ret.Model = &model
	}
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		var err error
		ret.Done, ret.Total, ret.Frames, err = sqlite.VectorProgress(ctx, model)
		return err
	})
	return ret, err
}

func (r *mutationResolver) RunSearchIndex(ctx context.Context, sceneIds []string) (string, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return "", err
	}
	ids, err := stringslice.StringSliceToIntSlice(sceneIds)
	if err != nil {
		return "", err
	}
	name := "Search index: scenes not indexed yet"
	if len(ids) > 0 {
		name = fmt.Sprintf("Search index: %d scene(s)", len(ids))
	}
	jobID := manager.GetInstance().JobManager.Add(context.WithoutCancel(access.WithPrincipal(ctx, nil)), name, &manager.SearchIndexJob{SceneIDs: ids})
	return strconv.Itoa(jobID), nil
}
