package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/models"
)

// PlaylistStore persists playlists.
type PlaylistStore struct{}

func NewPlaylistStore() *PlaylistStore { return &PlaylistStore{} }

type playlistRow struct {
	ID          int       `db:"id"`
	UserID      int       `db:"user_id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	WatchLater  bool      `db:"watch_later"`
	CreatedAt   Timestamp `db:"created_at"`
	UpdatedAt   Timestamp `db:"updated_at"`
}

func (r playlistRow) resolve() *models.Playlist {
	return &models.Playlist{
		ID:          r.ID,
		UserID:      r.UserID,
		Name:        r.Name,
		Description: r.Description,
		WatchLater:  r.WatchLater,
		CreatedAt:   r.CreatedAt.Timestamp,
		UpdatedAt:   r.UpdatedAt.Timestamp,
	}
}

const playlistColumns = "id, user_id, name, description, watch_later, created_at, updated_at"

// WatchLaterName is the name of every user's Watch later playlist.
const WatchLaterName = "Watch later"

func (s *PlaylistStore) Find(ctx context.Context, id int) (*models.Playlist, error) {
	var r playlistRow
	err := dbWrapper.Get(ctx, &r, "SELECT "+playlistColumns+" FROM playlists WHERE id = ?", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.resolve(), nil
}

func (s *PlaylistStore) FindByUser(ctx context.Context, userID int) ([]*models.Playlist, error) {
	var rows []playlistRow
	if err := dbWrapper.Select(ctx, &rows,
		"SELECT "+playlistColumns+" FROM playlists WHERE user_id = ? ORDER BY watch_later DESC, name COLLATE NOCASE ASC, id ASC", userID); err != nil {
		return nil, err
	}
	ret := make([]*models.Playlist, len(rows))
	for i, r := range rows {
		ret[i] = r.resolve()
	}
	return ret, nil
}

func (s *PlaylistStore) WatchLater(ctx context.Context, userID int) (*models.Playlist, error) {
	var r playlistRow
	err := dbWrapper.Get(ctx, &r, "SELECT "+playlistColumns+" FROM playlists WHERE user_id = ? AND watch_later = 1", userID)
	if err == nil {
		return r.resolve(), nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	p := &models.Playlist{UserID: userID, Name: WatchLaterName, WatchLater: true}
	if err := s.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *PlaylistStore) Create(ctx context.Context, p *models.Playlist) error {
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	res, err := dbWrapper.Exec(ctx,
		"INSERT INTO playlists (user_id, name, description, watch_later, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		p.UserID, p.Name, p.Description, p.WatchLater, Timestamp{now}, Timestamp{now})
	if err != nil {
		return fmt.Errorf("creating playlist: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = int(id)
	return nil
}

func (s *PlaylistStore) Update(ctx context.Context, p *models.Playlist) error {
	p.UpdatedAt = time.Now()
	_, err := dbWrapper.Exec(ctx, "UPDATE playlists SET name = ?, description = ?, updated_at = ? WHERE id = ?",
		p.Name, p.Description, Timestamp{p.UpdatedAt}, p.ID)
	return err
}

func (s *PlaylistStore) touch(ctx context.Context, id int) error {
	_, err := dbWrapper.Exec(ctx, "UPDATE playlists SET updated_at = ? WHERE id = ?", Timestamp{time.Now()}, id)
	return err
}

func (s *PlaylistStore) Destroy(ctx context.Context, id int) error {
	_, err := dbWrapper.Exec(ctx, "DELETE FROM playlists WHERE id = ?", id)
	return err
}

func (s *PlaylistStore) SceneIDs(ctx context.Context, id int) ([]int, error) {
	where, args := sceneAccessWhere(ctx, "playlists_scenes.scene_id")
	var ids []int
	if err := dbWrapper.Select(ctx, &ids,
		"SELECT playlists_scenes.scene_id FROM playlists_scenes WHERE playlists_scenes.playlist_id = ?"+where+" ORDER BY playlists_scenes.position ASC, playlists_scenes.added_at ASC",
		append([]interface{}{id}, args...)...); err != nil {
		return nil, err
	}
	return ids, nil
}

func (s *PlaylistStore) SceneCount(ctx context.Context, id int) (int, error) {
	where, args := sceneAccessWhere(ctx, "playlists_scenes.scene_id")
	var n int
	err := dbWrapper.Get(ctx, &n, "SELECT COUNT(*) FROM playlists_scenes WHERE playlists_scenes.playlist_id = ?"+where,
		append([]interface{}{id}, args...)...)
	return n, err
}

func (s *PlaylistStore) AddScenes(ctx context.Context, id int, sceneIDs []int) error {
	if len(sceneIDs) == 0 {
		return nil
	}
	var maxPos sql.NullInt64
	if err := dbWrapper.Get(ctx, &maxPos, "SELECT MAX(position) FROM playlists_scenes WHERE playlist_id = ?", id); err != nil {
		return err
	}
	pos := 0
	if maxPos.Valid {
		pos = int(maxPos.Int64) + 1
	}
	now := Timestamp{time.Now()}
	for _, sid := range sceneIDs {
		res, err := dbWrapper.Exec(ctx,
			"INSERT OR IGNORE INTO playlists_scenes (playlist_id, scene_id, position, added_at) SELECT ?, id, ?, ? FROM scenes WHERE id = ?",
			id, pos, now, sid)
		if err != nil {
			return fmt.Errorf("adding scene %d to playlist: %w", sid, err)
		}
		if n, _ := res.RowsAffected(); n > 0 {
			pos++
		}
	}
	return s.touch(ctx, id)
}

func (s *PlaylistStore) RemoveScenes(ctx context.Context, id int, sceneIDs []int) error {
	if len(sceneIDs) == 0 {
		return nil
	}
	q, args := inClause("DELETE FROM playlists_scenes WHERE playlist_id = ? AND scene_id IN", sceneIDs)
	if _, err := dbWrapper.Exec(ctx, q, append([]interface{}{id}, args...)...); err != nil {
		return err
	}
	return s.touch(ctx, id)
}

func (s *PlaylistStore) SetOrder(ctx context.Context, id int, sceneIDs []int) error {
	// scenes not in the list (for example ones the caller cannot see) keep
	// their relative order after the given ones
	if _, err := dbWrapper.Exec(ctx, "UPDATE playlists_scenes SET position = position + ? WHERE playlist_id = ?", len(sceneIDs)+1, id); err != nil {
		return err
	}
	for i, sid := range sceneIDs {
		if _, err := dbWrapper.Exec(ctx, "UPDATE playlists_scenes SET position = ? WHERE playlist_id = ? AND scene_id = ?", i, id, sid); err != nil {
			return err
		}
	}
	return s.touch(ctx, id)
}

func (s *PlaylistStore) PlaylistIDsWithScene(ctx context.Context, userID int, sceneID int) ([]int, error) {
	var ids []int
	err := dbWrapper.Select(ctx, &ids,
		"SELECT p.id FROM playlists p JOIN playlists_scenes ps ON ps.playlist_id = p.id WHERE p.user_id = ? AND ps.scene_id = ?",
		userID, sceneID)
	return ids, err
}

func (s *PlaylistStore) ContinueWatching(ctx context.Context, userID int, limit int) ([]int, error) {
	if limit <= 0 {
		limit = 20
	}

	resume := "scenes.resume_time"
	lastTouched := "scenes.updated_at"
	historyCond := "v.user_id IS NULL"
	if userID > 0 {
		resume = "(SELECT us.resume_time FROM users_scenes us WHERE us.user_id = ? AND us.scene_id = scenes.id)"
		lastTouched = "(SELECT us.updated_at FROM users_scenes us WHERE us.user_id = ? AND us.scene_id = scenes.id)"
		historyCond = "v.user_id = ?"
	}

	where, accessArgs := sceneAccessWhere(ctx, "scenes.id")

	q := fmt.Sprintf(`SELECT scenes.id FROM scenes
JOIN scenes_files sf ON sf.scene_id = scenes.id AND sf.`+"`primary`"+` = 1
JOIN video_files vf ON vf.file_id = sf.file_id
WHERE COALESCE(%[1]s, 0) > 30 AND COALESCE(%[1]s, 0) < vf.duration * 0.92%[2]s
ORDER BY COALESCE((SELECT MAX(v.view_date) FROM scenes_view_dates v WHERE v.scene_id = scenes.id AND %[3]s), %[4]s) DESC, scenes.id DESC
LIMIT ?`, resume, where, historyCond, lastTouched)

	var args []interface{}
	if userID > 0 {
		// the resume expression is used twice in WHERE
		args = append(args, userID, userID)
	}
	args = append(args, accessArgs...)
	if userID > 0 {
		args = append(args, userID, userID)
	}
	args = append(args, limit)

	var ids []int
	if err := dbWrapper.Select(ctx, &ids, q, args...); err != nil {
		return nil, err
	}
	return ids, nil
}

// PlaylistUserID returns the user id whose playlists the request uses: 0
// for the owner and for requests without a login.
func PlaylistUserID(ctx context.Context) int {
	if p := access.FromContext(ctx); p != nil {
		return p.UserID
	}
	return 0
}
