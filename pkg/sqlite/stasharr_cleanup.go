package sqlite

// Stasharr: cleanup suggestions, scenes that take space without being
// watched or liked.

import (
	"context"
	"fmt"
	"time"
)

// CleanupItem is a suggested scene with its size and history.
type CleanupItem struct {
	SceneID    int        `db:"scene_id"`
	Size       int64      `db:"size"`
	Plays      int        `db:"plays"`
	LastPlayed *Timestamp `db:"last_played"`
	Rating     *float64   `db:"rating"`
	AddedAt    Timestamp  `db:"added_at"`
}

// CleanupKinds: "unwatched" (not played by anyone in N months, added before
// that), "never" (never played, largest first), "low_rated" (2 stars or
// less from the owner or on average from everyone).
const cleanupBase = `
WITH sizes AS (
  SELECT sf.scene_id, SUM(f.size) AS size FROM scenes_files sf JOIN files f ON f.id = sf.file_id GROUP BY sf.scene_id
), plays AS (
  SELECT scene_id, COUNT(*) AS plays, MAX(view_date) AS last_played FROM scenes_view_dates GROUP BY scene_id
), ratings AS (
  SELECT scene_id, AVG(rating) AS rating FROM users_scenes WHERE rating IS NOT NULL GROUP BY scene_id
)
SELECT s.id AS scene_id, COALESCE(z.size, 0) AS size, COALESCE(p.plays, 0) AS plays, p.last_played,
  COALESCE(s.rating, r.rating) AS rating, s.created_at AS added_at
FROM scenes s
LEFT JOIN sizes z ON z.scene_id = s.id
LEFT JOIN plays p ON p.scene_id = s.id
LEFT JOIN ratings r ON r.scene_id = s.id
WHERE %s
ORDER BY %s`

// CleanupSuggestions returns matching scenes (biggest first) and their
// total count and size.
func CleanupSuggestions(ctx context.Context, kind string, months int, limit int) ([]CleanupItem, int, int64, error) {
	if months < 1 {
		months = 6
	}
	cut := Timestamp{time.Now().AddDate(0, -months, 0)}
	var where string
	var args []interface{}
	order := "size DESC"
	switch kind {
	case "never":
		where = "p.plays IS NULL AND s.created_at < ?"
		args = append(args, Timestamp{time.Now().AddDate(0, 0, -30)})
	case "low_rated":
		where = "COALESCE(s.rating, r.rating) IS NOT NULL AND COALESCE(s.rating, r.rating) <= 40"
		order = "rating ASC, size DESC"
	default: // unwatched
		where = "s.created_at < ? AND (p.last_played IS NULL OR p.last_played < ?)"
		args = append(args, cut, cut)
	}
	q := fmt.Sprintf(cleanupBase, where, order)

	var all []CleanupItem
	if err := dbWrapper.Select(ctx, &all, q, args...); err != nil {
		return nil, 0, 0, err
	}
	var total int64
	for _, it := range all {
		total += it.Size
	}
	n := len(all)
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all, n, total, nil
}
