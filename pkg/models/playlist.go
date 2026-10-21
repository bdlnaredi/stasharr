package models

import (
	"context"
	"time"
)

// Playlist is a user's ordered list of scenes. Every user, including the
// owner (UserID 0), has one Watch later playlist.
type Playlist struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	WatchLater  bool      `json:"watch_later"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PlaylistReaderWriter stores playlists and their scenes.
type PlaylistReaderWriter interface {
	Find(ctx context.Context, id int) (*Playlist, error)
	FindByUser(ctx context.Context, userID int) ([]*Playlist, error)
	// WatchLater returns the user's Watch later playlist, creating it if needed.
	WatchLater(ctx context.Context, userID int) (*Playlist, error)
	Create(ctx context.Context, p *Playlist) error
	Update(ctx context.Context, p *Playlist) error
	Destroy(ctx context.Context, id int) error

	// SceneIDs returns the scenes of a playlist in order, filtered by the
	// caller's access policy.
	SceneIDs(ctx context.Context, id int) ([]int, error)
	SceneCount(ctx context.Context, id int) (int, error)
	AddScenes(ctx context.Context, id int, sceneIDs []int) error
	RemoveScenes(ctx context.Context, id int, sceneIDs []int) error
	// SetOrder puts the given scenes first, in the given order.
	SetOrder(ctx context.Context, id int, sceneIDs []int) error
	// PlaylistIDsWithScene returns which of the user's playlists hold a scene.
	PlaylistIDsWithScene(ctx context.Context, userID int, sceneID int) ([]int, error)

	// ContinueWatching returns scenes the user started but did not finish,
	// most recently watched first.
	ContinueWatching(ctx context.Context, userID int, limit int) ([]int, error)
}
