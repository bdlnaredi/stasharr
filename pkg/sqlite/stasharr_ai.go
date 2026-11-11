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
	// what the suggestion's frames looked like, to learn from the review
	Model *string  `db:"model"`
	Scale *float64 `db:"scale"`
	Vec   []byte   `db:"vec"`
}

const aiSuggestionColumns = "id, scene_id, label, seconds, end_seconds, score, created_at, model, scale, vec"

// AIReplaceSuggestions swaps a scene's waiting suggestions for new ones and
// records that the scene was looked at.
func AIReplaceSuggestions(ctx context.Context, sceneID int, list []AIMarkerSuggestion, found int) error {
	if _, err := dbWrapper.Exec(ctx, "DELETE FROM ai_marker_suggestions WHERE scene_id = ?", sceneID); err != nil {
		return err
	}
	now := Timestamp{time.Now()}
	for _, s := range list {
		if _, err := dbWrapper.Exec(ctx,
			"INSERT INTO ai_marker_suggestions (scene_id, label, seconds, end_seconds, score, created_at, model, scale, vec) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			sceneID, s.Label, s.Seconds, s.EndSeconds, s.Score, now, s.Model, s.Scale, s.Vec); err != nil {
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

// AllSceneIDs lists every scene, newest first, to look at them all again.
func AllSceneIDs(ctx context.Context) ([]int, error) {
	var ids []int
	err := dbWrapper.Select(ctx, &ids, "SELECT id FROM scenes ORDER BY created_at DESC")
	return ids, err
}

// AIExample is a kept or thrown away suggestion the tagger learns from.
type AIExample struct {
	Label string  `db:"label"`
	Kept  bool    `db:"kept"`
	Model string  `db:"model"`
	Scale float64 `db:"scale"`
	Vec   []byte  `db:"vec"`
}

// AIAddExample records a reviewed suggestion.
func AIAddExample(ctx context.Context, sceneID int, e AIExample) error {
	_, err := dbWrapper.Exec(ctx,
		"INSERT INTO ai_marker_examples (label, kept, model, scale, vec, scene_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		e.Label, e.Kept, e.Model, e.Scale, e.Vec, sceneID, Timestamp{time.Now()})
	return err
}

// AIExamples returns the newest examples for a model, at most perLabel
// kept and perLabel thrown away for each label.
func AIExamples(ctx context.Context, model string, perLabel int) ([]AIExample, error) {
	var ret []AIExample
	err := dbWrapper.Select(ctx, &ret, `SELECT label, kept, model, scale, vec FROM (
  SELECT label, kept, model, scale, vec,
    ROW_NUMBER() OVER (PARTITION BY lower(label), kept ORDER BY id DESC) AS n
  FROM ai_marker_examples WHERE model = ?
) WHERE n <= ?`, model, perLabel)
	return ret, err
}

// AIExampleCount is how many examples a label has.
type AIExampleCount struct {
	Label  string `db:"label"`
	Kept   int    `db:"kept"`
	Thrown int    `db:"thrown"`
}

// AIExampleCounts counts the examples of each label.
func AIExampleCounts(ctx context.Context) ([]AIExampleCount, error) {
	var ret []AIExampleCount
	err := dbWrapper.Select(ctx, &ret, `SELECT label, SUM(CASE WHEN kept THEN 1 ELSE 0 END) AS kept, SUM(CASE WHEN kept THEN 0 ELSE 1 END) AS thrown
FROM ai_marker_examples GROUP BY lower(label)`)
	return ret, err
}

// AIForgetExamples deletes what was learned, for one label or all.
func AIForgetExamples(ctx context.Context, label string) error {
	if label == "" {
		_, err := dbWrapper.Exec(ctx, "DELETE FROM ai_marker_examples")
		return err
	}
	_, err := dbWrapper.Exec(ctx, "DELETE FROM ai_marker_examples WHERE lower(label) = lower(?)", label)
	return err
}

// FrameVectorsBetween returns a scene's search vectors from start to end
// seconds and the model that made them, for suggestions made before the
// tagger sent vectors.
func FrameVectorsBetween(ctx context.Context, sceneID int, start, end float64) (string, []FrameVector, error) {
	var model []string
	if err := dbWrapper.Select(ctx, &model, "SELECT model FROM scene_vector_runs WHERE scene_id = ?", sceneID); err != nil || len(model) == 0 {
		return "", nil, err
	}
	var list []FrameVector
	err := dbWrapper.Select(ctx, &list, "SELECT scene_id, seconds, scale, vec FROM scene_frame_vectors WHERE scene_id = ? AND seconds >= ? AND seconds <= ? ORDER BY seconds", sceneID, start-1, end+1)
	return model[0], list, err
}
