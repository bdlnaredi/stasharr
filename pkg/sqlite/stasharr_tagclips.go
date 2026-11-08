package sqlite

import (
	"context"
	"time"
)

// TagClip records which scene and moment a tag's clip was cut from.
type TagClip struct {
	TagID     int       `db:"tag_id"`
	SceneID   int       `db:"scene_id"`
	Seconds   float64   `db:"seconds"`
	CreatedAt Timestamp `db:"created_at"`
}

// TagClipCandidate is a scene that could give a tag its clip.
type TagClipCandidate struct {
	SceneID int      `db:"scene_id"`
	Marker  *float64 `db:"marker"`
}

// FindTagClip returns a tag's clip record, or nil.
func FindTagClip(ctx context.Context, tagID int) (*TagClip, error) {
	var ret []*TagClip
	if err := dbWrapper.Select(ctx, &ret, "SELECT tag_id, scene_id, seconds, created_at FROM tag_clips WHERE tag_id = ?", tagID); err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return nil, nil
	}
	return ret[0], nil
}

// SaveTagClip records a tag's clip.
func SaveTagClip(ctx context.Context, c TagClip) error {
	_, err := dbWrapper.Exec(ctx,
		"INSERT INTO tag_clips (tag_id, scene_id, seconds, created_at) VALUES (?, ?, ?, ?) ON CONFLICT(tag_id) DO UPDATE SET scene_id = excluded.scene_id, seconds = excluded.seconds, created_at = excluded.created_at",
		c.TagID, c.SceneID, c.Seconds, Timestamp{time.Now()})
	return err
}

// DeleteTagClip forgets a tag's clip.
func DeleteTagClip(ctx context.Context, tagID int) error {
	_, err := dbWrapper.Exec(ctx, "DELETE FROM tag_clips WHERE tag_id = ?", tagID)
	return err
}

const tagClipEligible = `tags.image_blob IS NULL AND (
  EXISTS (SELECT 1 FROM scenes_tags st WHERE st.tag_id = tags.id)
  OR EXISTS (SELECT 1 FROM scene_markers m WHERE m.primary_tag_id = tags.id))`

// TagIDsNeedingClips lists tags without their own picture that have scenes
// but no clip yet (or all of them with redo), busiest tags first.
func TagIDsNeedingClips(ctx context.Context, redo bool) ([]int, error) {
	q := "SELECT tags.id FROM tags WHERE " + tagClipEligible
	if !redo {
		q += " AND NOT EXISTS (SELECT 1 FROM tag_clips c WHERE c.tag_id = tags.id)"
	}
	q += " ORDER BY (SELECT COUNT(*) FROM scenes_tags st WHERE st.tag_id = tags.id) DESC, tags.id"
	var ids []int
	err := dbWrapper.Select(ctx, &ids, q)
	return ids, err
}

// TagClipProgress counts tags with a clip, out of the tags that can have one.
func TagClipProgress(ctx context.Context) (done, total int, err error) {
	if err = dbWrapper.Get(ctx, &total, "SELECT COUNT(*) FROM tags WHERE "+tagClipEligible); err != nil {
		return
	}
	err = dbWrapper.Get(ctx, &done, "SELECT COUNT(*) FROM tags JOIN tag_clips c ON c.tag_id = tags.id WHERE "+tagClipEligible)
	return
}

// TagClipCandidates lists the best scenes for a tag's clip: scenes with a
// marker for the tag first (with the marker's time), then by rating, then
// by how often they were played.
func TagClipCandidates(ctx context.Context, tagID, limit int) ([]TagClipCandidate, error) {
	const q = `
SELECT s.id AS scene_id,
  (SELECT m.seconds FROM scene_markers m
     WHERE m.scene_id = s.id AND (m.primary_tag_id = ?1
       OR EXISTS (SELECT 1 FROM scene_markers_tags mt WHERE mt.scene_marker_id = m.id AND mt.tag_id = ?1))
     ORDER BY m.seconds LIMIT 1) AS marker
FROM scenes s
WHERE EXISTS (SELECT 1 FROM scenes_tags st WHERE st.scene_id = s.id AND st.tag_id = ?1)
   OR EXISTS (SELECT 1 FROM scene_markers m2 WHERE m2.scene_id = s.id AND m2.primary_tag_id = ?1)
ORDER BY (marker IS NOT NULL) DESC,
  COALESCE(s.rating, 0) DESC,
  (SELECT COUNT(*) FROM scenes_view_dates v WHERE v.scene_id = s.id) DESC,
  s.id DESC
LIMIT ?2`
	var ret []TagClipCandidate
	err := dbWrapper.Select(ctx, &ret, q, tagID, limit)
	return ret, err
}
