package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/models"
)

// UserStore persists additional user accounts and tiers.
type UserStore struct{}

func NewUserStore() *UserStore { return &UserStore{} }

type userRow struct {
	ID           int            `db:"id"`
	Username     string         `db:"username"`
	PasswordHash string         `db:"password_hash"`
	Role         string         `db:"role"`
	TierID       sql.NullInt64  `db:"tier_id"`
	APIKey       sql.NullString `db:"api_key"`
	Disabled     bool           `db:"disabled"`
	LastLoginAt  NullTimestamp  `db:"last_login_at"`
	CreatedAt    Timestamp      `db:"created_at"`
	UpdatedAt    Timestamp      `db:"updated_at"`
}

func (r userRow) resolve() *models.User {
	u := &models.User{
		ID:           r.ID,
		Username:     r.Username,
		PasswordHash: r.PasswordHash,
		Role:         access.Role(r.Role),
		Disabled:     r.Disabled,
		LastLoginAt:  r.LastLoginAt.TimePtr(),
		CreatedAt:    r.CreatedAt.Timestamp,
		UpdatedAt:    r.UpdatedAt.Timestamp,
	}
	if r.TierID.Valid {
		id := int(r.TierID.Int64)
		u.TierID = &id
	}
	if r.APIKey.Valid {
		k := r.APIKey.String
		u.APIKey = &k
	}
	return u
}

const userColumns = "id, username, password_hash, role, tier_id, api_key, disabled, last_login_at, created_at, updated_at"

func (s *UserStore) getOne(ctx context.Context, where string, args ...interface{}) (*models.User, error) {
	var r userRow
	err := dbWrapper.Get(ctx, &r, "SELECT "+userColumns+" FROM users WHERE "+where, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.resolve(), nil
}

func (s *UserStore) Find(ctx context.Context, id int) (*models.User, error) {
	return s.getOne(ctx, "id = ?", id)
}

func (s *UserStore) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	return s.getOne(ctx, "username = ? COLLATE NOCASE", username)
}

func (s *UserStore) FindByAPIKey(ctx context.Context, key string) (*models.User, error) {
	if key == "" {
		return nil, nil
	}
	return s.getOne(ctx, "api_key = ?", key)
}

func (s *UserStore) All(ctx context.Context) ([]*models.User, error) {
	var rows []userRow
	if err := dbWrapper.Select(ctx, &rows, "SELECT "+userColumns+" FROM users ORDER BY username COLLATE NOCASE"); err != nil {
		return nil, err
	}
	ret := make([]*models.User, len(rows))
	for i, r := range rows {
		ret[i] = r.resolve()
	}
	return ret, nil
}

func (s *UserStore) Count(ctx context.Context) (int, error) {
	var n int
	err := dbWrapper.Get(ctx, &n, "SELECT COUNT(*) FROM users")
	return n, err
}

func nullInt(p *int) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

func nullString(p *string) interface{} {
	if p == nil || *p == "" {
		return nil
	}
	return *p
}

func (s *UserStore) Create(ctx context.Context, u *models.User) error {
	now := time.Now()
	u.CreatedAt, u.UpdatedAt = now, now
	res, err := dbWrapper.Exec(ctx,
		"INSERT INTO users (username, password_hash, role, tier_id, api_key, disabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		u.Username, u.PasswordHash, string(u.Role), nullInt(u.TierID), nullString(u.APIKey), u.Disabled, Timestamp{now}, Timestamp{now})
	if err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	u.ID = int(id)
	return nil
}

func (s *UserStore) Update(ctx context.Context, u *models.User) error {
	u.UpdatedAt = time.Now()
	_, err := dbWrapper.Exec(ctx,
		"UPDATE users SET username = ?, password_hash = ?, role = ?, tier_id = ?, api_key = ?, disabled = ?, updated_at = ? WHERE id = ?",
		u.Username, u.PasswordHash, string(u.Role), nullInt(u.TierID), nullString(u.APIKey), u.Disabled, Timestamp{u.UpdatedAt}, u.ID)
	if err != nil {
		return fmt.Errorf("updating user: %w", err)
	}
	return nil
}

func (s *UserStore) TouchLogin(ctx context.Context, id int) error {
	_, err := dbWrapper.Exec(ctx, "UPDATE users SET last_login_at = ? WHERE id = ?", Timestamp{time.Now()}, id)
	return err
}

func (s *UserStore) Destroy(ctx context.Context, id int) error {
	// playlists are keyed by user id without a foreign key, as the owner is 0
	if _, err := dbWrapper.Exec(ctx, "DELETE FROM playlists WHERE user_id = ?", id); err != nil {
		return err
	}
	_, err := dbWrapper.Exec(ctx, "DELETE FROM users WHERE id = ?", id)
	return err
}

// --- tiers

type tierRow struct {
	ID             int            `db:"id"`
	Name           string         `db:"name"`
	Description    sql.NullString `db:"description"`
	AllowedPaths   string         `db:"allowed_paths"`
	BlockedTagIDs  string         `db:"blocked_tag_ids"`
	RequiredTagIDs string         `db:"required_tag_ids"`
	CanDownload    bool           `db:"can_download"`
	CanViewReels   bool           `db:"can_view_reels"`
	CreatedAt      Timestamp      `db:"created_at"`
	UpdatedAt      Timestamp      `db:"updated_at"`
}

func (r tierRow) resolve() (*models.UserTier, error) {
	t := &models.UserTier{
		ID:           r.ID,
		Name:         r.Name,
		Description:  r.Description.String,
		CanDownload:  r.CanDownload,
		CanViewReels: r.CanViewReels,
		CreatedAt:    r.CreatedAt.Timestamp,
		UpdatedAt:    r.UpdatedAt.Timestamp,
	}
	if err := json.Unmarshal([]byte(r.AllowedPaths), &t.AllowedPaths); err != nil {
		return nil, fmt.Errorf("tier %d allowed_paths: %w", r.ID, err)
	}
	if err := json.Unmarshal([]byte(r.BlockedTagIDs), &t.BlockedTagIDs); err != nil {
		return nil, fmt.Errorf("tier %d blocked_tag_ids: %w", r.ID, err)
	}
	if err := json.Unmarshal([]byte(r.RequiredTagIDs), &t.RequiredTagIDs); err != nil {
		return nil, fmt.Errorf("tier %d required_tag_ids: %w", r.ID, err)
	}
	return t, nil
}

const tierColumns = "id, name, description, allowed_paths, blocked_tag_ids, required_tag_ids, can_download, can_view_reels, created_at, updated_at"

func jsonOrEmpty(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil || string(b) == "null" {
		return "[]"
	}
	return string(b)
}

func (s *UserStore) FindTier(ctx context.Context, id int) (*models.UserTier, error) {
	var r tierRow
	err := dbWrapper.Get(ctx, &r, "SELECT "+tierColumns+" FROM user_tiers WHERE id = ?", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.resolve()
}

func (s *UserStore) AllTiers(ctx context.Context) ([]*models.UserTier, error) {
	var rows []tierRow
	if err := dbWrapper.Select(ctx, &rows, "SELECT "+tierColumns+" FROM user_tiers ORDER BY name COLLATE NOCASE"); err != nil {
		return nil, err
	}
	ret := make([]*models.UserTier, 0, len(rows))
	for _, r := range rows {
		t, err := r.resolve()
		if err != nil {
			return nil, err
		}
		ret = append(ret, t)
	}
	return ret, nil
}

func (s *UserStore) CreateTier(ctx context.Context, t *models.UserTier) error {
	now := time.Now()
	t.CreatedAt, t.UpdatedAt = now, now
	res, err := dbWrapper.Exec(ctx,
		"INSERT INTO user_tiers (name, description, allowed_paths, blocked_tag_ids, required_tag_ids, can_download, can_view_reels, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		t.Name, t.Description, jsonOrEmpty(t.AllowedPaths), jsonOrEmpty(t.BlockedTagIDs), jsonOrEmpty(t.RequiredTagIDs), t.CanDownload, t.CanViewReels, Timestamp{now}, Timestamp{now})
	if err != nil {
		return fmt.Errorf("creating tier: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	t.ID = int(id)
	return nil
}

func (s *UserStore) UpdateTier(ctx context.Context, t *models.UserTier) error {
	t.UpdatedAt = time.Now()
	_, err := dbWrapper.Exec(ctx,
		"UPDATE user_tiers SET name = ?, description = ?, allowed_paths = ?, blocked_tag_ids = ?, required_tag_ids = ?, can_download = ?, can_view_reels = ?, updated_at = ? WHERE id = ?",
		t.Name, t.Description, jsonOrEmpty(t.AllowedPaths), jsonOrEmpty(t.BlockedTagIDs), jsonOrEmpty(t.RequiredTagIDs), t.CanDownload, t.CanViewReels, Timestamp{t.UpdatedAt}, t.ID)
	if err != nil {
		return fmt.Errorf("updating tier: %w", err)
	}
	return nil
}

func (s *UserStore) DestroyTier(ctx context.Context, id int) error {
	_, err := dbWrapper.Exec(ctx, "DELETE FROM user_tiers WHERE id = ?", id)
	return err
}

// --- per-user scene data

func (s *UserStore) GetSceneData(ctx context.Context, userID int, sceneIDs []int) (map[int]models.UserSceneData, error) {
	ret := make(map[int]models.UserSceneData, len(sceneIDs))
	if len(sceneIDs) == 0 {
		return ret, nil
	}
	type row struct {
		SceneID      int           `db:"scene_id"`
		Rating       sql.NullInt64 `db:"rating"`
		ResumeTime   float64       `db:"resume_time"`
		PlayDuration float64       `db:"play_duration"`
	}
	err := batchExec(sceneIDs, defaultBatchSize, func(batch []int) error {
		query, args := inClause("SELECT scene_id, rating, resume_time, play_duration FROM users_scenes WHERE user_id = ? AND scene_id IN", batch)
		var rows []row
		if err := dbWrapper.Select(ctx, &rows, query, append([]interface{}{userID}, args...)...); err != nil {
			return err
		}
		for _, r := range rows {
			d := models.UserSceneData{UserID: userID, SceneID: r.SceneID, ResumeTime: r.ResumeTime, PlayDuration: r.PlayDuration}
			if r.Rating.Valid {
				v := int(r.Rating.Int64)
				d.Rating = &v
			}
			ret[r.SceneID] = d
		}
		return nil
	})
	return ret, err
}

func inClause(prefix string, ids []int) (string, []interface{}) {
	args := make([]interface{}, len(ids))
	q := prefix + " ("
	for i, id := range ids {
		if i > 0 {
			q += ","
		}
		q += "?"
		args[i] = id
	}
	return q + ")", args
}

func (s *UserStore) ensureSceneRow(ctx context.Context, userID, sceneID int) error {
	_, err := dbWrapper.Exec(ctx,
		"INSERT OR IGNORE INTO users_scenes (user_id, scene_id, updated_at) VALUES (?, ?, ?)",
		userID, sceneID, Timestamp{time.Now()})
	return err
}

func (s *UserStore) SetSceneRating(ctx context.Context, userID, sceneID int, rating *int) error {
	if err := s.ensureSceneRow(ctx, userID, sceneID); err != nil {
		return err
	}
	_, err := dbWrapper.Exec(ctx, "UPDATE users_scenes SET rating = ?, updated_at = ? WHERE user_id = ? AND scene_id = ?",
		nullInt(rating), Timestamp{time.Now()}, userID, sceneID)
	return err
}

// SaveSceneActivity updates resume time and adds to play duration.
func (s *UserStore) SaveSceneActivity(ctx context.Context, userID, sceneID int, resumeTime, playDuration *float64) error {
	if err := s.ensureSceneRow(ctx, userID, sceneID); err != nil {
		return err
	}
	if resumeTime != nil {
		if _, err := dbWrapper.Exec(ctx, "UPDATE users_scenes SET resume_time = ?, updated_at = ? WHERE user_id = ? AND scene_id = ?",
			*resumeTime, Timestamp{time.Now()}, userID, sceneID); err != nil {
			return err
		}
	}
	if playDuration != nil && *playDuration > 0 {
		if _, err := dbWrapper.Exec(ctx, "UPDATE users_scenes SET play_duration = play_duration + ?, updated_at = ? WHERE user_id = ? AND scene_id = ?",
			*playDuration, Timestamp{time.Now()}, userID, sceneID); err != nil {
			return err
		}
	}
	return nil
}

func (s *UserStore) ResetSceneActivity(ctx context.Context, userID, sceneID int, resetResume, resetDuration bool) error {
	if resetResume {
		if _, err := dbWrapper.Exec(ctx, "UPDATE users_scenes SET resume_time = 0 WHERE user_id = ? AND scene_id = ?", userID, sceneID); err != nil {
			return err
		}
	}
	if resetDuration {
		if _, err := dbWrapper.Exec(ctx, "UPDATE users_scenes SET play_duration = 0 WHERE user_id = ? AND scene_id = ?", userID, sceneID); err != nil {
			return err
		}
	}
	return nil
}

func (s *UserStore) TotalPlayDuration(ctx context.Context, userID int) (float64, error) {
	var total sql.NullFloat64
	err := dbWrapper.Get(ctx, &total, "SELECT SUM(play_duration) FROM users_scenes WHERE user_id = ?", userID)
	return total.Float64, err
}
