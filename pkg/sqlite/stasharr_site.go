package sqlite

// Stasharr site front end: scene lists counted across every user. Ratings
// and plays are kept per user, so a viewer's own "most played" or "top
// rated" would be empty; the site shows what is popular with everyone.

import (
	"context"
	"fmt"
	"time"
)

// SiteSceneSort picks how a site list is ordered.
type SiteSceneSort int

const (
	SitePopular SiteSceneSort = iota
	SiteTrending
	SiteTopRated
	SiteRecent
	SiteRandom
)

// trendingWindow is how far back the trending list looks.
const trendingWindow = 14 * 24 * time.Hour

// SiteSceneFilter narrows a site list. Zero values mean no filter.
type SiteSceneFilter struct {
	TagID       int
	PerformerID int
}

// SiteSceneIDs returns a page of scene ids in the given order, and how many
// scenes the list holds.
func SiteSceneIDs(ctx context.Context, sort SiteSceneSort, filter SiteSceneFilter, limit, offset int) ([]int, int, error) {
	tagID := filter.TagID
	where, args := sceneAccessWhere(ctx, "scenes.id")
	var joins string
	var joinArgs []interface{}
	cond := "1=1"
	var order string

	switch sort {
	case SitePopular, SiteTrending:
		since := ""
		if sort == SiteTrending {
			since = " WHERE view_date >= ?"
			joinArgs = append(joinArgs, Timestamp{time.Now().Add(-trendingWindow)})
		}
		joins = "LEFT JOIN (SELECT scene_id, COUNT(*) AS c FROM scenes_view_dates" + since + " GROUP BY scene_id) v ON v.scene_id = scenes.id"
		order = "COALESCE(v.c, 0) DESC, scenes.created_at DESC, scenes.id DESC"
	case SiteTopRated:
		// the average of every rating: the owner's and each user's
		joins = "LEFT JOIN (SELECT scene_id, SUM(rating) AS s, COUNT(rating) AS n FROM users_scenes WHERE rating IS NOT NULL GROUP BY scene_id) u ON u.scene_id = scenes.id"
		cond = "(scenes.rating IS NOT NULL OR COALESCE(u.n, 0) > 0)"
		order = "(COALESCE(scenes.rating, 0) + COALESCE(u.s, 0)) * 1.0 / ((scenes.rating IS NOT NULL) + COALESCE(u.n, 0)) DESC, ((scenes.rating IS NOT NULL) + COALESCE(u.n, 0)) DESC, scenes.id DESC"
	case SiteRandom:
		order = "RANDOM()"
	default:
		order = "scenes.created_at DESC, scenes.id DESC"
	}

	if tagID > 0 {
		cond += " AND EXISTS (SELECT 1 FROM scenes_tags st WHERE st.scene_id = scenes.id AND st.tag_id = ?)"
	}
	if filter.PerformerID > 0 {
		cond += " AND EXISTS (SELECT 1 FROM performers_scenes ps WHERE ps.scene_id = scenes.id AND ps.performer_id = ?)"
	}

	from := "FROM scenes " + joins + " WHERE " + cond + where
	qArgs := append([]interface{}{}, joinArgs...)
	if tagID > 0 {
		qArgs = append(qArgs, tagID)
	}
	if filter.PerformerID > 0 {
		qArgs = append(qArgs, filter.PerformerID)
	}
	qArgs = append(qArgs, args...)

	var total int
	if err := dbWrapper.Get(ctx, &total, "SELECT COUNT(*) "+from, qArgs...); err != nil {
		return nil, 0, fmt.Errorf("counting scenes: %w", err)
	}

	var ids []int
	q := "SELECT scenes.id " + from + " ORDER BY " + order + " LIMIT ? OFFSET ?"
	if err := dbWrapper.Select(ctx, &ids, q, append(qArgs, limit, offset)...); err != nil {
		return nil, 0, fmt.Errorf("listing scenes: %w", err)
	}
	return ids, total, nil
}

// ---------------------------------------------------------------- favourites

// Kinds of site favourites.
const (
	SiteLikedScene    = "scene"
	SiteFavoriteModel = "performer"
)

// SiteToggleFavorite adds or removes a favourite and reports whether it is
// now a favourite.
func SiteToggleFavorite(ctx context.Context, userID int, kind string, id int) (bool, error) {
	res, err := dbWrapper.Exec(ctx, "DELETE FROM site_favorites WHERE user_id = ? AND kind = ? AND target_id = ?", userID, kind, id)
	if err != nil {
		return false, err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return false, nil
	}
	if _, err := dbWrapper.Exec(ctx, "INSERT INTO site_favorites (user_id, kind, target_id, created_at) VALUES (?, ?, ?, ?)",
		userID, kind, id, Timestamp{time.Now()}); err != nil {
		return false, err
	}
	return true, nil
}

// SiteIsFavorite reports whether the user has liked or favourited an item.
func SiteIsFavorite(ctx context.Context, userID int, kind string, id int) (bool, error) {
	var n int
	err := dbWrapper.Get(ctx, &n, "SELECT COUNT(*) FROM site_favorites WHERE user_id = ? AND kind = ? AND target_id = ?", userID, kind, id)
	return n > 0, err
}

// SiteLikedSceneIDs returns the user's liked scenes, most recent first.
func SiteLikedSceneIDs(ctx context.Context, userID int, limit, offset int) ([]int, int, error) {
	where, args := sceneAccessWhere(ctx, "scenes.id")
	from := "FROM site_favorites f JOIN scenes ON scenes.id = f.target_id WHERE f.user_id = ? AND f.kind = ?" + where
	qArgs := append([]interface{}{userID, SiteLikedScene}, args...)
	var total int
	if err := dbWrapper.Get(ctx, &total, "SELECT COUNT(*) "+from, qArgs...); err != nil {
		return nil, 0, err
	}
	var ids []int
	err := dbWrapper.Select(ctx, &ids, "SELECT scenes.id "+from+" ORDER BY f.created_at DESC LIMIT ? OFFSET ?", append(qArgs, limit, offset)...)
	return ids, total, err
}

// SiteFavoriteModelIDs returns the user's favourite performers, most recent first.
func SiteFavoriteModelIDs(ctx context.Context, userID int) ([]int, error) {
	var ids []int
	err := dbWrapper.Select(ctx, &ids,
		"SELECT f.target_id FROM site_favorites f JOIN performers p ON p.id = f.target_id WHERE f.user_id = ? AND f.kind = ? ORDER BY f.created_at DESC",
		userID, SiteFavoriteModel)
	return ids, err
}

// SiteFromFavoritesIDs returns the newest scenes with any of the user's
// favourite performers.
func SiteFromFavoritesIDs(ctx context.Context, userID int, limit, offset int) ([]int, int, error) {
	where, args := sceneAccessWhere(ctx, "scenes.id")
	from := `FROM scenes WHERE EXISTS (SELECT 1 FROM performers_scenes ps JOIN site_favorites f ON f.target_id = ps.performer_id AND f.kind = ? AND f.user_id = ? WHERE ps.scene_id = scenes.id)` + where
	qArgs := append([]interface{}{SiteFavoriteModel, userID}, args...)
	var total int
	if err := dbWrapper.Get(ctx, &total, "SELECT COUNT(*) "+from, qArgs...); err != nil {
		return nil, 0, err
	}
	var ids []int
	err := dbWrapper.Select(ctx, &ids, "SELECT scenes.id "+from+" ORDER BY scenes.created_at DESC, scenes.id DESC LIMIT ? OFFSET ?", append(qArgs, limit, offset)...)
	return ids, total, err
}

// ---------------------------------------------------------------- ratings and co-stars

// SiteRatingSummary returns the average rating of a scene across the owner
// and every user, and how many ratings there are.
func SiteRatingSummary(ctx context.Context, sceneID int) (float64, int, error) {
	var row struct {
		Sum   float64 `db:"s"`
		Count int     `db:"n"`
	}
	err := dbWrapper.Get(ctx, &row, `SELECT COALESCE(SUM(r), 0) AS s, COUNT(r) AS n FROM (
SELECT rating AS r FROM scenes WHERE id = ? AND rating IS NOT NULL
UNION ALL
SELECT rating AS r FROM users_scenes WHERE scene_id = ? AND rating IS NOT NULL)`, sceneID, sceneID)
	if err != nil || row.Count == 0 {
		return 0, 0, err
	}
	return row.Sum / float64(row.Count), row.Count, nil
}

// SiteCoStar is a performer who appears in scenes with another one.
type SiteCoStar struct {
	ID     int `db:"id"`
	Scenes int `db:"c"`
}

// SiteCoStars returns the performers who most often share a scene with the
// given one, counting only scenes the user can see.
func SiteCoStars(ctx context.Context, performerID int, limit int) ([]SiteCoStar, error) {
	where, args := sceneAccessWhere(ctx, "a.scene_id")
	q := `SELECT b.performer_id AS id, COUNT(*) AS c FROM performers_scenes a
JOIN performers_scenes b ON b.scene_id = a.scene_id AND b.performer_id != a.performer_id
WHERE a.performer_id = ?` + where + `
GROUP BY b.performer_id ORDER BY c DESC, b.performer_id LIMIT ?`
	var rows []SiteCoStar
	err := dbWrapper.Select(ctx, &rows, q, append(append([]interface{}{performerID}, args...), limit)...)
	return rows, err
}
