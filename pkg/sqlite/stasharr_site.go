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

// SiteSceneIDs returns a page of scene ids in the given order, and how many
// scenes the list holds. tagID, when above 0, keeps scenes with that tag.
func SiteSceneIDs(ctx context.Context, sort SiteSceneSort, tagID int, limit, offset int) ([]int, int, error) {
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

	from := "FROM scenes " + joins + " WHERE " + cond + where
	qArgs := append([]interface{}{}, joinArgs...)
	if tagID > 0 {
		qArgs = append(qArgs, tagID)
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
