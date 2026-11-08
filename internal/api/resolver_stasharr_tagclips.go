package api

import (
	"context"
	"fmt"
	"strconv"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stashapp/stash/pkg/sqlite"
)

func (r *tagResolver) Clip(ctx context.Context, obj *models.Tag) (*TagClip, error) {
	var ret *TagClip
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		c, err := sqlite.FindTagClip(ctx, obj.ID)
		if err != nil || c == nil {
			return err
		}
		// only for viewers who may see the scene it was cut from
		s, err := r.repository.Scene.Find(ctx, c.SceneID)
		if err != nil || s == nil {
			return err
		}
		baseURL, _ := ctx.Value(BaseURLCtxKey).(string)
		v := c.CreatedAt.Timestamp.Unix()
		ret = &TagClip{
			Video:   fmt.Sprintf("%s/tag/%d/clip.mp4?t=%d", baseURL, obj.ID, v),
			Still:   fmt.Sprintf("%s/tag/%d/clip.jpg?t=%d", baseURL, obj.ID, v),
			SceneID: strconv.Itoa(c.SceneID),
			Seconds: c.Seconds,
		}
		return nil
	})
	return ret, err
}

func (r *queryResolver) TagClipProgress(ctx context.Context) (*TagClipProgress, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	ret := &TagClipProgress{}
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		var err error
		ret.Done, ret.Total, err = sqlite.TagClipProgress(ctx)
		return err
	})
	return ret, err
}

func (r *mutationResolver) MakeTagClips(ctx context.Context, tagIds []string, redo *bool) (string, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return "", err
	}
	ids, err := stringslice.StringSliceToIntSlice(tagIds)
	if err != nil {
		return "", err
	}
	j := &manager.TagClipJob{TagIDs: ids, Redo: redo != nil && *redo}
	name := "Tag clips: tags without one"
	switch {
	case len(ids) == 1:
		name = "Tag clip: tag " + tagIds[0]
	case len(ids) > 1:
		name = fmt.Sprintf("Tag clips: %d tags", len(ids))
	case j.Redo:
		name = "Tag clips: remake all"
	}
	// jobs see the whole library; clips are only shown to viewers who may
	// see the scene they come from
	jobID := manager.GetInstance().JobManager.Add(context.WithoutCancel(access.WithPrincipal(ctx, nil)), name, j)
	return strconv.Itoa(jobID), nil
}
