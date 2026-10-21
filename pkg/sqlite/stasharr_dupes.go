package sqlite

import (
	"context"
	"fmt"
)

// MergeSceneHistory moves the play and O history, per-user ratings and
// resume positions, playlist entries, markers and reels of one scene onto
// another, before the first scene is deleted as a duplicate. Markers and
// reels are only moved when the destination has none of its own.
func MergeSceneHistory(ctx context.Context, fromID, toID int) error {
	stmts := []struct {
		q    string
		args []interface{}
	}{
		{"UPDATE scenes_view_dates SET scene_id = ? WHERE scene_id = ?", []interface{}{toID, fromID}},
		{"UPDATE scenes_o_dates SET scene_id = ? WHERE scene_id = ?", []interface{}{toID, fromID}},
		// keep the destination's own row for a user if it has one
		{"UPDATE OR IGNORE users_scenes SET scene_id = ? WHERE scene_id = ?", []interface{}{toID, fromID}},
		{"UPDATE OR IGNORE playlists_scenes SET scene_id = ? WHERE scene_id = ?", []interface{}{toID, fromID}},
		{"UPDATE scenes SET play_duration = play_duration + COALESCE((SELECT s2.play_duration FROM scenes s2 WHERE s2.id = ?), 0) WHERE id = ?", []interface{}{fromID, toID}},
		{"UPDATE scenes SET resume_time = (SELECT s2.resume_time FROM scenes s2 WHERE s2.id = ?) WHERE id = ? AND resume_time = 0", []interface{}{fromID, toID}},
		{"UPDATE scene_markers SET scene_id = ? WHERE scene_id = ? AND NOT EXISTS (SELECT 1 FROM scene_markers m2 WHERE m2.scene_id = ?)", []interface{}{toID, fromID, toID}},
		{"UPDATE reels SET scene_id = ? WHERE scene_id = ? AND NOT EXISTS (SELECT 1 FROM reels r2 WHERE r2.scene_id = ?)", []interface{}{toID, fromID, toID}},
	}
	for _, s := range stmts {
		if _, err := dbWrapper.Exec(ctx, s.q, s.args...); err != nil {
			return fmt.Errorf("moving history from scene %d to %d: %w", fromID, toID, err)
		}
	}
	return nil
}
