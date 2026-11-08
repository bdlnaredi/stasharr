package sqlite

import (
	"context"
	"time"
)

// Library-wide numbers for the admin stats page.

// StatsTotals are the library's totals.
type StatsTotals struct {
	Scenes     int     `db:"scenes"`
	Duration   float64 `db:"duration"`
	Size       float64 `db:"size"`
	Performers int     `db:"performers"`
	Tags       int     `db:"tags"`
	Studios    int     `db:"studios"`
	Reels      int     `db:"reels"`
	ReelViews  int     `db:"reel_views"`
	Comments   int     `db:"comments"`
	Users      int     `db:"users"`
	Plays      int     `db:"plays"`
}

// StatsTotalsQuery counts the library.
func StatsTotalsQuery(ctx context.Context) (*StatsTotals, error) {
	ret := &StatsTotals{}
	err := dbWrapper.Get(ctx, ret, `
SELECT
  (SELECT COUNT(*) FROM scenes) AS scenes,
  (SELECT COALESCE(SUM(vf.duration), 0) FROM scenes_files sf JOIN video_files vf ON vf.file_id = sf.file_id WHERE sf."primary" = 1) AS duration,
  (SELECT COALESCE(SUM(f.size), 0) FROM scenes_files sf JOIN files f ON f.id = sf.file_id) AS size,
  (SELECT COUNT(*) FROM performers) AS performers,
  (SELECT COUNT(*) FROM tags) AS tags,
  (SELECT COUNT(*) FROM studios) AS studios,
  (SELECT COUNT(*) FROM reels) AS reels,
  (SELECT COALESCE(SUM(views), 0) FROM reels) AS reel_views,
  (SELECT COUNT(*) FROM site_comments) AS comments,
  (SELECT COUNT(*) FROM users WHERE disabled = 0) AS users,
  (SELECT COUNT(*) FROM scenes_view_dates) AS plays`)
	return ret, err
}

// StatsMonth is one month of additions.
type StatsMonth struct {
	Month  string  `db:"month"`
	Scenes int     `db:"scenes"`
	Size   float64 `db:"size"`
}

// StatsAddedByMonth lists scenes added per month, oldest first.
func StatsAddedByMonth(ctx context.Context, months int) ([]StatsMonth, error) {
	since := time.Now().AddDate(0, -(months - 1), 0)
	since = time.Date(since.Year(), since.Month(), 1, 0, 0, 0, 0, time.Local)
	var ret []StatsMonth
	err := dbWrapper.Select(ctx, &ret, `
SELECT strftime('%Y-%m', datetime(s.created_at)) AS month,
  COUNT(DISTINCT s.id) AS scenes,
  COALESCE(SUM(f.size), 0) AS size
FROM scenes s
LEFT JOIN scenes_files sf ON sf.scene_id = s.id AND sf."primary" = 1
LEFT JOIN files f ON f.id = sf.file_id
WHERE datetime(s.created_at) >= datetime(?)
GROUP BY month ORDER BY month`, Timestamp{since})
	return ret, err
}

// StatsCount is a name and a number.
type StatsCount struct {
	ID    int    `db:"id"`
	Name  string `db:"name"`
	Count int    `db:"count"`
}

// StatsResolutions counts scenes by picture size.
func StatsResolutions(ctx context.Context) ([]StatsCount, error) {
	var ret []StatsCount
	err := dbWrapper.Select(ctx, &ret, `
SELECT 0 AS id, CASE
    WHEN MIN(vf.width, vf.height) >= 2000 THEN '4K and up'
    WHEN MIN(vf.width, vf.height) >= 1000 THEN '1080p'
    WHEN MIN(vf.width, vf.height) >= 700 THEN '720p'
    ELSE 'Below 720p' END AS name,
  COUNT(*) AS count
FROM scenes_files sf JOIN video_files vf ON vf.file_id = sf.file_id
WHERE sf."primary" = 1
GROUP BY name ORDER BY MIN(MIN(vf.width, vf.height)) DESC`)
	return ret, err
}

// StatsTopScenes lists the most played scenes since a time.
func StatsTopScenes(ctx context.Context, since time.Time, limit int) ([]StatsCount, error) {
	var ret []StatsCount
	err := dbWrapper.Select(ctx, &ret, `
SELECT v.scene_id AS id, '' AS name, COUNT(*) AS count
FROM scenes_view_dates v
WHERE datetime(v.view_date) >= datetime(?)
GROUP BY v.scene_id ORDER BY count DESC, v.scene_id DESC LIMIT ?`, Timestamp{since}, limit)
	return ret, err
}

// StatsTopPerformers lists the performers whose scenes were played most.
func StatsTopPerformers(ctx context.Context, since time.Time, limit int) ([]StatsCount, error) {
	var ret []StatsCount
	err := dbWrapper.Select(ctx, &ret, `
SELECT p.id AS id, p.name AS name, COUNT(*) AS count
FROM scenes_view_dates v
JOIN performers_scenes ps ON ps.scene_id = v.scene_id
JOIN performers p ON p.id = ps.performer_id
WHERE datetime(v.view_date) >= datetime(?)
GROUP BY p.id ORDER BY count DESC, p.name LIMIT ?`, Timestamp{since}, limit)
	return ret, err
}

// StatsTopTags lists the tags whose scenes were played most.
func StatsTopTags(ctx context.Context, since time.Time, limit int) ([]StatsCount, error) {
	var ret []StatsCount
	err := dbWrapper.Select(ctx, &ret, `
SELECT t.id AS id, t.name AS name, COUNT(*) AS count
FROM scenes_view_dates v
JOIN scenes_tags st ON st.scene_id = v.scene_id
JOIN tags t ON t.id = st.tag_id
WHERE datetime(v.view_date) >= datetime(?)
GROUP BY t.id ORDER BY count DESC, t.name LIMIT ?`, Timestamp{since}, limit)
	return ret, err
}

// StatsTopReels lists the most watched reels (all time).
func StatsTopReels(ctx context.Context, limit int) ([]StatsCount, error) {
	var ret []StatsCount
	err := dbWrapper.Select(ctx, &ret, `
SELECT r.id AS id, CAST(r.scene_id AS TEXT) AS name, r.views AS count
FROM reels r WHERE r.views > 0 ORDER BY r.views DESC, r.id DESC LIMIT ?`, limit)
	return ret, err
}

// StatsDay is one day's plays.
type StatsDay struct {
	Day   string `db:"day"`
	Plays int    `db:"plays"`
}

// StatsPlaysByDay counts plays per day since a time.
func StatsPlaysByDay(ctx context.Context, since time.Time) ([]StatsDay, error) {
	var ret []StatsDay
	err := dbWrapper.Select(ctx, &ret, `
SELECT date(datetime(view_date, 'localtime')) AS day, COUNT(*) AS plays
FROM scenes_view_dates WHERE datetime(view_date) >= datetime(?)
GROUP BY day ORDER BY day`, Timestamp{since})
	return ret, err
}

// StatsUser is one account's activity. UserID 0 is the owner.
type StatsUser struct {
	UserID    int     `db:"user_id"`
	Name      string  `db:"name"`
	Role      string  `db:"role"`
	Plays     int     `db:"plays"`
	LastPlay  *string `db:"last_play"`
	LastLogin *string `db:"last_login"`
}

// StatsUsers lists every account's plays since a time and when they last
// played and signed in. The owner has no row in users.
func StatsUsers(ctx context.Context, since time.Time) ([]StatsUser, error) {
	var ret []StatsUser
	err := dbWrapper.Select(ctx, &ret, `
SELECT 0 AS user_id, '' AS name, 'OWNER' AS role,
  (SELECT COUNT(*) FROM scenes_view_dates v WHERE v.user_id IS NULL AND datetime(v.view_date) >= datetime(?1)) AS plays,
  (SELECT CAST(MAX(v.view_date) AS TEXT) FROM scenes_view_dates v WHERE v.user_id IS NULL) AS last_play,
  NULL AS last_login
UNION ALL
SELECT u.id, u.username, u.role,
  (SELECT COUNT(*) FROM scenes_view_dates v WHERE v.user_id = u.id AND datetime(v.view_date) >= datetime(?1)),
  (SELECT CAST(MAX(v.view_date) AS TEXT) FROM scenes_view_dates v WHERE v.user_id = u.id),
  CAST(u.last_login_at AS TEXT)
FROM users u WHERE u.disabled = 0
ORDER BY plays DESC`, Timestamp{since})
	return ret, err
}

// StatsPlay is one recent play.
type StatsPlay struct {
	UserID  int    `db:"user_id"`
	Name    string `db:"name"`
	SceneID int    `db:"scene_id"`
	At      string `db:"at"`
}

// StatsRecentPlays lists the latest plays by anyone.
func StatsRecentPlays(ctx context.Context, limit int) ([]StatsPlay, error) {
	var ret []StatsPlay
	err := dbWrapper.Select(ctx, &ret, `
SELECT COALESCE(v.user_id, 0) AS user_id, COALESCE(u.username, '') AS name, v.scene_id AS scene_id, CAST(v.view_date AS TEXT) AS at
FROM scenes_view_dates v LEFT JOIN users u ON u.id = v.user_id
ORDER BY datetime(v.view_date) DESC LIMIT ?`, limit)
	return ret, err
}
