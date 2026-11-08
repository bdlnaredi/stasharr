package sqlite

import (
	"context"
	"time"
)

// FrameVector is one sampled frame's search vector.
type FrameVector struct {
	SceneID int     `db:"scene_id"`
	Seconds float64 `db:"seconds"`
	Scale   float64 `db:"scale"`
	Vec     []byte  `db:"vec"`
}

// SaveFrameVectors replaces a scene's frame vectors and records the model.
func SaveFrameVectors(ctx context.Context, sceneID int, model string, list []FrameVector) error {
	if _, err := dbWrapper.Exec(ctx, "DELETE FROM scene_frame_vectors WHERE scene_id = ?", sceneID); err != nil {
		return err
	}
	for _, v := range list {
		if _, err := dbWrapper.Exec(ctx,
			"INSERT OR REPLACE INTO scene_frame_vectors (scene_id, seconds, scale, vec) VALUES (?, ?, ?, ?)",
			sceneID, v.Seconds, v.Scale, v.Vec); err != nil {
			return err
		}
	}
	_, err := dbWrapper.Exec(ctx,
		"INSERT INTO scene_vector_runs (scene_id, model, frames, ran_at) VALUES (?, ?, ?, ?) ON CONFLICT(scene_id) DO UPDATE SET model = excluded.model, frames = excluded.frames, ran_at = excluded.ran_at",
		sceneID, model, len(list), Timestamp{time.Now()})
	return err
}

// SceneIDsToIndex lists scenes not indexed with this model yet, newest first.
func SceneIDsToIndex(ctx context.Context, model string) ([]int, error) {
	var ids []int
	err := dbWrapper.Select(ctx, &ids, `SELECT s.id FROM scenes s
WHERE NOT EXISTS (SELECT 1 FROM scene_vector_runs r WHERE r.scene_id = s.id AND r.model = ?)
ORDER BY s.created_at DESC`, model)
	return ids, err
}

// VectorProgress counts scenes indexed with this model, out of all scenes.
func VectorProgress(ctx context.Context, model string) (done, total, frames int, err error) {
	if err = dbWrapper.Get(ctx, &done, "SELECT COUNT(*) FROM scene_vector_runs WHERE model = ?", model); err != nil {
		return
	}
	if err = dbWrapper.Get(ctx, &frames, "SELECT COALESCE(SUM(frames), 0) FROM scene_vector_runs WHERE model = ?", model); err != nil {
		return
	}
	err = dbWrapper.Get(ctx, &total, "SELECT COUNT(*) FROM scenes")
	return
}

// AllFrameVectors loads every frame vector made with this model, a scene at
// a time through fn, to build the search index in memory.
func AllFrameVectors(ctx context.Context, model string, fn func(FrameVector)) error {
	rows, err := dbWrapper.Queryx(ctx, `SELECT v.scene_id, v.seconds, v.scale, v.vec FROM scene_frame_vectors v
JOIN scene_vector_runs r ON r.scene_id = v.scene_id AND r.model = ?
ORDER BY v.scene_id, v.seconds`, model)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var v FrameVector
		if err := rows.StructScan(&v); err != nil {
			return err
		}
		fn(v)
	}
	return rows.Err()
}
