package sqlite

// Stasharr: profiles under one account, their PINs, watch time per day and
// the monthly recap.

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"time"
)

// ---------------------------------------------------------------- profiles

// Profile ties a user account to the main account it belongs to.
// ParentID 0 is the owner account.
type Profile struct {
	UserID   int    `db:"user_id"`
	ParentID int    `db:"parent_id"`
	Name     string `db:"name"`
	Color    string `db:"color"`
}

// FindProfile returns the profile row of a user account, or nil when the
// account is a main account.
func FindProfile(ctx context.Context, userID int) (*Profile, error) {
	var p Profile
	err := dbWrapper.Get(ctx, &p, "SELECT user_id, parent_id, name, color FROM user_profiles WHERE user_id = ?", userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ProfilesOf lists the profiles of a main account, oldest first.
func ProfilesOf(ctx context.Context, parentID int) ([]Profile, error) {
	var ret []Profile
	err := dbWrapper.Select(ctx, &ret, "SELECT user_id, parent_id, name, color FROM user_profiles WHERE parent_id = ? ORDER BY created_at, user_id", parentID)
	return ret, err
}

// ProfileUserIDs lists every account that is a profile, to hide them from
// the admin's user list.
func ProfileUserIDs(ctx context.Context) (map[int]bool, error) {
	var ids []int
	if err := dbWrapper.Select(ctx, &ids, "SELECT user_id FROM user_profiles"); err != nil {
		return nil, err
	}
	ret := map[int]bool{}
	for _, id := range ids {
		ret[id] = true
	}
	return ret, nil
}

func CreateProfile(ctx context.Context, p Profile) error {
	_, err := dbWrapper.Exec(ctx, "INSERT INTO user_profiles (user_id, parent_id, name, color, created_at) VALUES (?, ?, ?, ?, ?)",
		p.UserID, p.ParentID, p.Name, p.Color, Timestamp{time.Now()})
	return err
}

func UpdateProfile(ctx context.Context, p Profile) error {
	_, err := dbWrapper.Exec(ctx, "UPDATE user_profiles SET name = ?, color = ? WHERE user_id = ?", p.Name, p.Color, p.UserID)
	return err
}

// ---------------------------------------------------------------- PINs

// FindPin returns the PIN hash of an account (0 = owner), or "".
func FindPin(ctx context.Context, accountID int) (string, error) {
	var h string
	err := dbWrapper.Get(ctx, &h, "SELECT pin_hash FROM profile_pins WHERE account_id = ?", accountID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return h, err
}

// SetPin saves a PIN hash, or removes the PIN when hash is "".
func SetPin(ctx context.Context, accountID int, hash string) error {
	if hash == "" {
		_, err := dbWrapper.Exec(ctx, "DELETE FROM profile_pins WHERE account_id = ?", accountID)
		return err
	}
	_, err := dbWrapper.Exec(ctx, `INSERT INTO profile_pins (account_id, pin_hash) VALUES (?, ?)
ON CONFLICT(account_id) DO UPDATE SET pin_hash = excluded.pin_hash`, accountID, hash)
	return err
}

// ---------------------------------------------------------------- watch time

// AddWatchTime adds seconds watched today to an account (0 = owner).
func AddWatchTime(ctx context.Context, userID int, seconds float64) error {
	if seconds <= 0 || seconds > 6*3600 {
		return nil
	}
	day := time.Now().Format("2006-01-02")
	_, err := dbWrapper.Exec(ctx, `INSERT INTO user_watch_days (user_id, day, seconds) VALUES (?, ?, ?)
ON CONFLICT(user_id, day) DO UPDATE SET seconds = seconds + excluded.seconds`, userID, day, seconds)
	return err
}

// ---------------------------------------------------------------- recap

// RecapCount is a name with how many plays it had.
type RecapCount struct {
	ID    int    `db:"id"`
	Name  string `db:"name"`
	Plays int    `db:"plays"`
}

// Recap is an account's month in numbers.
type Recap struct {
	Month        string
	Plays        int
	Scenes       int
	Seconds      float64
	ActiveDays   int
	LongestRun   int
	Likes        int
	Models       []RecapCount
	Tags         []RecapCount
	Studios      []RecapCount
	TopSceneID   int
	TopScenePlay int
	// plays per hour of the day (0-23) and per weekday (0 = Sunday)
	Hours    [24]int
	Weekdays [7]int
}

// historyWhere picks the account's rows in scenes_view_dates: the owner's
// are the ones without a user.
func historyWhere(userID int) (string, []interface{}) {
	if userID == 0 {
		return "v.user_id IS NULL", nil
	}
	return "v.user_id = ?", []interface{}{userID}
}

// RecapMonths lists the months (YYYY-MM) with plays, newest first.
func RecapMonths(ctx context.Context, userID int, limit int) ([]string, error) {
	where, args := historyWhere(userID)
	var ret []string
	err := dbWrapper.Select(ctx, &ret, "SELECT DISTINCT substr(v.view_date, 1, 7) AS m FROM scenes_view_dates v WHERE "+where+" ORDER BY m DESC LIMIT ?", append(args, limit)...)
	return ret, err
}

// MonthRecap works out an account's recap for one month (YYYY-MM).
func MonthRecap(ctx context.Context, userID int, month string) (*Recap, error) {
	where, args := historyWhere(userID)
	where += " AND substr(v.view_date, 1, 7) = ?"
	args = append(args, month)
	r := &Recap{Month: month}

	if err := dbWrapper.Get(ctx, &r.Plays, "SELECT COUNT(*) FROM scenes_view_dates v WHERE "+where, args...); err != nil {
		return nil, err
	}
	if err := dbWrapper.Get(ctx, &r.Scenes, "SELECT COUNT(DISTINCT v.scene_id) FROM scenes_view_dates v WHERE "+where, args...); err != nil {
		return nil, err
	}
	var secs sql.NullFloat64
	if err := dbWrapper.Get(ctx, &secs, "SELECT SUM(seconds) FROM user_watch_days WHERE user_id = ? AND substr(day, 1, 7) = ?", userID, month); err != nil {
		return nil, err
	}
	r.Seconds = secs.Float64
	if err := dbWrapper.Get(ctx, &r.Likes, "SELECT COUNT(*) FROM site_favorites WHERE user_id = ? AND kind = ? AND substr(created_at, 1, 7) = ?", userID, SiteLikedScene, month); err != nil {
		return nil, err
	}

	if err := dbWrapper.Select(ctx, &r.Models, `SELECT p.id AS id, p.name AS name, COUNT(*) AS plays
FROM scenes_view_dates v JOIN performers_scenes ps ON ps.scene_id = v.scene_id JOIN performers p ON p.id = ps.performer_id
WHERE `+where+` GROUP BY p.id ORDER BY plays DESC, p.name LIMIT 5`, args...); err != nil {
		return nil, err
	}
	if err := dbWrapper.Select(ctx, &r.Tags, `SELECT t.id AS id, t.name AS name, COUNT(*) AS plays
FROM scenes_view_dates v JOIN scenes_tags st ON st.scene_id = v.scene_id JOIN tags t ON t.id = st.tag_id
WHERE `+where+` GROUP BY t.id ORDER BY plays DESC, t.name LIMIT 6`, args...); err != nil {
		return nil, err
	}
	if err := dbWrapper.Select(ctx, &r.Studios, `SELECT st.id AS id, st.name AS name, COUNT(*) AS plays
FROM scenes_view_dates v JOIN scenes s ON s.id = v.scene_id JOIN studios st ON st.id = s.studio_id
WHERE `+where+` GROUP BY st.id ORDER BY plays DESC, st.name LIMIT 3`, args...); err != nil {
		return nil, err
	}

	var top []struct {
		SceneID int `db:"scene_id"`
		Plays   int `db:"plays"`
	}
	if err := dbWrapper.Select(ctx, &top, "SELECT v.scene_id AS scene_id, COUNT(*) AS plays FROM scenes_view_dates v WHERE "+where+" GROUP BY v.scene_id ORDER BY plays DESC LIMIT 1", args...); err != nil {
		return nil, err
	}
	if len(top) == 1 {
		r.TopSceneID, r.TopScenePlay = top[0].SceneID, top[0].Plays
	}

	// when: hours and weekdays from the stored local times
	var when []string
	if err := dbWrapper.Select(ctx, &when, "SELECT v.view_date FROM scenes_view_dates v WHERE "+where, args...); err != nil {
		return nil, err
	}
	days := map[string]bool{}
	for _, w := range when {
		t, err := time.Parse(time.RFC3339, w)
		if err != nil {
			continue
		}
		r.Hours[t.Hour()]++
		r.Weekdays[int(t.Weekday())]++
		days[t.Format("2006-01-02")] = true
	}
	r.ActiveDays = len(days)
	r.LongestRun = longestRun(days)
	return r, nil
}

// longestRun is the most days in a row in the set.
func longestRun(days map[string]bool) int {
	list := make([]string, 0, len(days))
	for d := range days {
		list = append(list, d)
	}
	sort.Strings(list)
	best, run := 0, 0
	var prev time.Time
	for i, d := range list {
		t, err := time.Parse("2006-01-02", d)
		if err != nil {
			continue
		}
		if i > 0 && t.Sub(prev) == 24*time.Hour {
			run++
		} else {
			run = 1
		}
		if run > best {
			best = run
		}
		prev = t
	}
	return best
}

// ForgetAccountData removes an account's likes, alerts, lists and watch time,
// when a profile is deleted. Comments stay, under the name they were
// written with.
func ForgetAccountData(ctx context.Context, userID int) error {
	for _, q := range []string{
		"DELETE FROM site_favorites WHERE user_id = ?",
		"DELETE FROM site_alerts_seen WHERE user_id = ?",
		"DELETE FROM user_watch_days WHERE user_id = ?",
		"DELETE FROM playlists_scenes WHERE playlist_id IN (SELECT id FROM playlists WHERE user_id = ?)",
		"DELETE FROM playlists WHERE user_id = ?",
	} {
		if _, err := dbWrapper.Exec(ctx, q, userID); err != nil {
			return err
		}
	}
	return nil
}
