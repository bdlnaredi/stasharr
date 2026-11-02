package sqlite

import (
	"context"
	"time"
)

// AIMarkerSuggestion is a marker the tagger suggests, waiting for review.
type AIMarkerSuggestion struct {
	ID         int       `db:"id"`
	SceneID    int       `db:"scene_id"`
	Label      string    `db:"label"`
	Seconds    float64   `db:"seconds"`
	EndSeconds float64   `db:"end_seconds"`
	Score      float64   `db:"score"`
	CreatedAt  Timestamp `db:"created_at"`
}

const aiSuggestionColumns = "id, scene_id, label, seconds, end_seconds, score, created_at"

// AIReplaceSuggestions swaps a scene's waiting suggestions for new ones and
// records that the scene was looked at.
func AIReplaceSuggestions(ctx context.Context, sceneID int, list []AIMarkerSuggestion, found int) error {
	if _, err := dbWrapper.Exec(ctx, "DELETE FROM ai_marker_suggestions WHERE scene_id = ?", sceneID); err != nil {
		return err
	}
	now := Timestamp{time.Now()}
	for _, s := range list {
		if _, err := dbWrapper.Exec(ctx,
			"INSERT INTO ai_marker_suggestions (scene_id, label, seconds, end_seconds, score, created_at) VALUES (?, ?, ?, ?, ?, ?)",
			sceneID, s.Label, s.Seconds, s.EndSeconds, s.Score, now); err != nil {
			return err
		}
	}
	_, err := dbWrapper.Exec(ctx,
		"INSERT INTO ai_marker_runs (scene_id, ran_at, found) VALUES (?, ?, ?) ON CONFLICT(scene_id) DO UPDATE SET ran_at = excluded.ran_at, found = excluded.found",
		sceneID, now, found)
	return err
}

// AISceneIDsToDo lists scenes the tagger has not looked at yet.
func AISceneIDsToDo(ctx context.Context) ([]int, error) {
	var ids []int
	err := dbWrapper.Select(ctx, &ids, "SELECT scenes.id FROM scenes WHERE NOT EXISTS (SELECT 1 FROM ai_marker_runs r WHERE r.scene_id = scenes.id) ORDER BY scenes.created_at DESC")
	return ids, err
}

// AIProgress is how many scenes were looked at, out of all scenes.
func AIProgress(ctx context.Context) (done, total int, err error) {
	if err = dbWrapper.Get(ctx, &done, "SELECT COUNT(*) FROM ai_marker_runs"); err != nil {
		return
	}
	err = dbWrapper.Get(ctx, &total, "SELECT COUNT(*) FROM scenes")
	return
}

// AISuggestions lists waiting suggestions, newest scenes first, and counts
// them and the scenes they belong to.
func AISuggestions(ctx context.Context, limit, offset int) (list []*AIMarkerSuggestion, total, scenes int, err error) {
	if err = dbWrapper.Get(ctx, &total, "SELECT COUNT(*) FROM ai_marker_suggestions"); err != nil {
		return
	}
	if err = dbWrapper.Get(ctx, &scenes, "SELECT COUNT(DISTINCT scene_id) FROM ai_marker_suggestions"); err != nil {
		return
	}
	err = dbWrapper.Select(ctx, &list, "SELECT "+aiSuggestionColumns+" FROM ai_marker_suggestions ORDER BY created_at DESC, scene_id DESC, seconds ASC LIMIT ? OFFSET ?", limit, offset)
	return
}

// AIFindSuggestions returns the suggestions with these ids.
func AIFindSuggestions(ctx context.Context, ids []int) ([]*AIMarkerSuggestion, error) {
	var ret []*AIMarkerSuggestion
	for _, id := range ids {
		var s []*AIMarkerSuggestion
		if err := dbWrapper.Select(ctx, &s, "SELECT "+aiSuggestionColumns+" FROM ai_marker_suggestions WHERE id = ?", id); err != nil {
			return nil, err
		}
		ret = append(ret, s...)
	}
	return ret, nil
}

// AIDeleteSuggestion removes one suggestion.
func AIDeleteSuggestion(ctx context.Context, id int) error {
	_, err := dbWrapper.Exec(ctx, "DELETE FROM ai_marker_suggestions WHERE id = ?", id)
	return err
}
