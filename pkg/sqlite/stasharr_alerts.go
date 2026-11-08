package sqlite

import (
	"context"
	"time"
)

// Alerts for new scenes with a user's favourite pornstars. A scene is new
// for a user when it was added after they last looked at their alerts (or
// in the last week, before they ever looked).

const alertsFirstLook = 7 * 24 * time.Hour

// AlertsSeenAt returns when the user last looked at their alerts.
func AlertsSeenAt(ctx context.Context, userID int) (time.Time, error) {
	var at []string
	if err := dbWrapper.Select(ctx, &at, "SELECT CAST(seen_at AS TEXT) FROM site_alerts_seen WHERE user_id = ?", userID); err != nil {
		return time.Time{}, err
	}
	if len(at) > 0 {
		if t, ok := ParseStatsTime(at[0]); ok {
			return t, nil
		}
	}
	return time.Now().Add(-alertsFirstLook), nil
}

// MarkAlertsSeen records that the user looked at their alerts now.
func MarkAlertsSeen(ctx context.Context, userID int) error {
	_, err := dbWrapper.Exec(ctx,
		"INSERT INTO site_alerts_seen (user_id, seen_at) VALUES (?, ?) ON CONFLICT(user_id) DO UPDATE SET seen_at = excluded.seen_at",
		userID, Timestamp{time.Now()})
	return err
}

// AlertScene is a new scene and the favourite in it.
type AlertScene struct {
	SceneID     int    `db:"scene_id"`
	PerformerID int    `db:"performer_id"`
	Performer   string `db:"performer"`
	CreatedAt   string `db:"created_at"`
}

// alertFavourites: the user's favourite pornstars on the site, and for the
// owner also the performers marked favourite in the library
const alertFavourites = `(
  SELECT f.target_id AS performer_id FROM site_favorites f WHERE f.user_id = ?1 AND f.kind = 'performer'
  UNION SELECT p.id FROM performers p WHERE ?1 = 0 AND p.favorite = 1)`

// AlertScenes lists scenes added since a time that have one of the user's
// favourites, newest first, with the first favourite in each.
func AlertScenes(ctx context.Context, userID int, since time.Time, limit int) ([]AlertScene, error) {
	var ret []AlertScene
	err := dbWrapper.Select(ctx, &ret, `
SELECT x.scene_id, x.performer_id, p.name AS performer, x.created_at FROM (
  SELECT s.id AS scene_id, MIN(ps.performer_id) AS performer_id, CAST(s.created_at AS TEXT) AS created_at
  FROM scenes s
  JOIN performers_scenes ps ON ps.scene_id = s.id
  WHERE ps.performer_id IN `+alertFavourites+`
    AND datetime(s.created_at) > datetime(?2)
  GROUP BY s.id
) x JOIN performers p ON p.id = x.performer_id
ORDER BY datetime(x.created_at) DESC, x.scene_id DESC
LIMIT ?3`, userID, Timestamp{since}, limit)
	return ret, err
}
