package api

import (
	"context"
	"fmt"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sqlite"
)

func recommendLimit(limit *int) int {
	if limit != nil && *limit > 0 && *limit <= 100 {
		return *limit
	}
	return 20
}

// loadRecommended loads scenes in ranked order, dropping ones the user's
// tier hides, and trims to n.
func (r *queryResolver) loadRecommended(ctx context.Context, ids []int, n int) ([]*models.Scene, error) {
	scenes, err := r.scenesInOrder(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(scenes) > n {
		scenes = scenes[:n]
	}
	return scenes, nil
}

func (r *queryResolver) FindSimilarScenes(ctx context.Context, sceneID string, limit *int) ([]*models.Scene, error) {
	id, err := strconv.Atoi(sceneID)
	if err != nil {
		return nil, fmt.Errorf("invalid id %q", sceneID)
	}
	n := recommendLimit(limit)
	ret := []*models.Scene{}
	err = r.withReadTxn(ctx, func(ctx context.Context) error {
		// the seed scene must be visible to the user
		seed, err := r.scenesInOrder(ctx, []int{id})
		if err != nil || len(seed) == 0 {
			return err
		}
		// ask for extra, as some may be hidden by the user's tier
		ids, err := sqlite.SimilarSceneIDs(ctx, id, n*3)
		if err != nil {
			return err
		}
		ret, err = r.loadRecommended(ctx, ids, n)
		return err
	})
	if ret == nil {
		ret = []*models.Scene{}
	}
	return ret, err
}

func (r *queryResolver) RecommendedScenes(ctx context.Context, limit *int) ([]*models.Scene, error) {
	n := recommendLimit(limit)
	ret := []*models.Scene{}
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ids, err := sqlite.RecommendedSceneIDs(ctx, n*3)
		if err != nil {
			return err
		}
		ret, err = r.loadRecommended(ctx, ids, n)
		return err
	})
	if ret == nil {
		ret = []*models.Scene{}
	}
	return ret, err
}
