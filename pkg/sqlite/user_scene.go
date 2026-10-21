package sqlite

import (
	"context"

	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/models"
)

var userStore = NewUserStore()

// overlayUserSceneData replaces the global rating and resume data with the
// current user's personal values. The owner keeps the global values.
func overlayUserSceneData(ctx context.Context, scenes []*models.Scene) error {
	uid := access.HistoryUserID(ctx)
	if uid == nil || len(scenes) == 0 {
		return nil
	}

	ids := make([]int, len(scenes))
	for i, s := range scenes {
		ids[i] = s.ID
	}

	data, err := userStore.GetSceneData(ctx, *uid, ids)
	if err != nil {
		return err
	}

	for _, s := range scenes {
		d := data[s.ID]
		s.Rating = d.Rating
		s.ResumeTime = d.ResumeTime
		s.PlayDuration = d.PlayDuration
	}
	return nil
}

// divertUserScenePartial stores personal fields of a partial update in the
// user's own table and removes them from the partial.
func divertUserScenePartial(ctx context.Context, sceneID int, partial *models.ScenePartial) error {
	uid := access.HistoryUserID(ctx)
	if uid == nil {
		return nil
	}

	if partial.Rating.Set {
		if err := userStore.SetSceneRating(ctx, *uid, sceneID, partial.Rating.Ptr()); err != nil {
			return err
		}
		partial.Rating = models.OptionalInt{}
	}

	if partial.ResumeTime.Set || partial.PlayDuration.Set {
		var resume *float64
		if partial.ResumeTime.Set {
			v := partial.ResumeTime.Value
			resume = &v
		}
		if err := userStore.SaveSceneActivity(ctx, *uid, sceneID, resume, nil); err != nil {
			return err
		}
		if partial.PlayDuration.Set {
			if err := userStore.ResetSceneActivity(ctx, *uid, sceneID, false, true); err != nil {
				return err
			}
			v := partial.PlayDuration.Value
			if err := userStore.SaveSceneActivity(ctx, *uid, sceneID, nil, &v); err != nil {
				return err
			}
		}
		partial.ResumeTime = models.OptionalFloat64{}
		partial.PlayDuration = models.OptionalFloat64{}
	}

	return nil
}

// divertUserSceneUpdate stores the personal fields of a full update for the
// current user and restores the global values in s before it is written.
func divertUserSceneUpdate(ctx context.Context, s *models.Scene) error {
	uid := access.HistoryUserID(ctx)
	if uid == nil {
		return nil
	}

	if err := userStore.SetSceneRating(ctx, *uid, s.ID, s.Rating); err != nil {
		return err
	}
	resume := s.ResumeTime
	if err := userStore.SaveSceneActivity(ctx, *uid, s.ID, &resume, nil); err != nil {
		return err
	}

	var global struct {
		Rating       *int    `db:"rating"`
		ResumeTime   float64 `db:"resume_time"`
		PlayDuration float64 `db:"play_duration"`
	}
	if err := dbWrapper.Get(ctx, &global, "SELECT rating, resume_time, play_duration FROM scenes WHERE id = ?", s.ID); err != nil {
		return err
	}
	s.Rating = global.Rating
	s.ResumeTime = global.ResumeTime
	s.PlayDuration = global.PlayDuration
	return nil
}
