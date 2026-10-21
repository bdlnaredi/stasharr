package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/stashapp/stash/pkg/models"
)

// ReelStore persists generated reels.
type ReelStore struct{}

func NewReelStore() *ReelStore { return &ReelStore{} }

type reelRow struct {
	ID        int       `db:"id"`
	SceneID   int       `db:"scene_id"`
	Path      string    `db:"path"`
	Duration  float64   `db:"duration"`
	Width     int       `db:"width"`
	Height    int       `db:"height"`
	Source    string    `db:"source"`
	Segments  string    `db:"segments"`
	CreatedAt Timestamp `db:"created_at"`
}

func (r reelRow) resolve() *models.Reel {
	ret := &models.Reel{
		ID:        r.ID,
		SceneID:   r.SceneID,
		Path:      r.Path,
		Duration:  r.Duration,
		Width:     r.Width,
		Height:    r.Height,
		Source:    r.Source,
		CreatedAt: r.CreatedAt.Timestamp,
	}
	_ = json.Unmarshal([]byte(r.Segments), &ret.Segments)
	return ret
}

const reelColumns = "reels.id, reels.scene_id, reels.path, reels.duration, reels.width, reels.height, reels.source, reels.segments, reels.created_at"

func (s *ReelStore) Create(ctx context.Context, r *models.Reel) error {
	r.CreatedAt = time.Now()
	segs, err := json.Marshal(r.Segments)
	if err != nil {
		return fmt.Errorf("encoding reel segments: %w", err)
	}
	res, err := dbWrapper.Exec(ctx,
		"INSERT INTO reels (scene_id, path, duration, width, height, source, segments, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		r.SceneID, r.Path, r.Duration, r.Width, r.Height, r.Source, string(segs), Timestamp{r.CreatedAt})
	if err != nil {
		return fmt.Errorf("creating reel: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	r.ID = int(id)
	return nil
}

func (s *ReelStore) Find(ctx context.Context, id int) (*models.Reel, error) {
	where, args := sceneAccessWhere(ctx, "reels.scene_id")
	var r reelRow
	err := dbWrapper.Get(ctx, &r, "SELECT "+reelColumns+" FROM reels WHERE reels.id = ?"+where, append([]interface{}{id}, args...)...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.resolve(), nil
}

func (s *ReelStore) FindBySceneID(ctx context.Context, sceneID int) ([]*models.Reel, error) {
	where, args := sceneAccessWhere(ctx, "reels.scene_id")
	var rows []reelRow
	if err := dbWrapper.Select(ctx, &rows, "SELECT "+reelColumns+" FROM reels WHERE reels.scene_id = ?"+where+" ORDER BY reels.created_at DESC",
		append([]interface{}{sceneID}, args...)...); err != nil {
		return nil, err
	}
	return resolveReels(rows), nil
}

// Query returns reels newest first, filtered by the caller's access policy.
func (s *ReelStore) Query(ctx context.Context, page, perPage int) ([]*models.Reel, int, error) {
	where, args := sceneAccessWhere(ctx, "reels.scene_id")
	var total int
	if err := dbWrapper.Get(ctx, &total, "SELECT COUNT(*) FROM reels WHERE 1=1"+where, args...); err != nil {
		return nil, 0, err
	}
	if perPage <= 0 {
		perPage = 40
	}
	if page <= 0 {
		page = 1
	}
	var rows []reelRow
	q := "SELECT " + reelColumns + " FROM reels WHERE 1=1" + where + " ORDER BY reels.created_at DESC, reels.id DESC LIMIT ? OFFSET ?"
	if err := dbWrapper.Select(ctx, &rows, q, append(args, perPage, (page-1)*perPage)...); err != nil {
		return nil, 0, err
	}
	return resolveReels(rows), total, nil
}

func (s *ReelStore) All(ctx context.Context) ([]*models.Reel, error) {
	var rows []reelRow
	if err := dbWrapper.Select(ctx, &rows, "SELECT "+reelColumns+" FROM reels"); err != nil {
		return nil, err
	}
	return resolveReels(rows), nil
}

func resolveReels(rows []reelRow) []*models.Reel {
	ret := make([]*models.Reel, len(rows))
	for i, r := range rows {
		ret[i] = r.resolve()
	}
	return ret
}

func (s *ReelStore) Destroy(ctx context.Context, id int) error {
	_, err := dbWrapper.Exec(ctx, "DELETE FROM reels WHERE id = ?", id)
	return err
}
