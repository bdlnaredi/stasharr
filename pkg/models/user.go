package models

import (
	"context"
	"time"

	"github.com/stashapp/stash/pkg/access"
)

// User is an additional login account stored in the database. The owner
// account configured in Settings > Security is not stored here.
type User struct {
	ID           int         `json:"id"`
	Username     string      `json:"username"`
	PasswordHash string      `json:"-"`
	Role         access.Role `json:"role"`
	TierID       *int        `json:"tier_id"`
	APIKey       *string     `json:"-"`
	Disabled     bool        `json:"disabled"`
	LastLoginAt  *time.Time  `json:"last_login_at"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// UserTier is a named library access policy that users are assigned to.
type UserTier struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	AllowedPaths   []string  `json:"allowed_paths"`
	BlockedTagIDs  []int     `json:"blocked_tag_ids"`
	RequiredTagIDs []int     `json:"required_tag_ids"`
	CanDownload    bool      `json:"can_download"`
	CanViewReels   bool      `json:"can_view_reels"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Policy converts the tier into an access policy.
func (t *UserTier) Policy() access.Policy {
	if t == nil {
		return access.Policy{CanDownload: true, CanViewReels: true}
	}
	return access.Policy{
		AllowedPaths:   t.AllowedPaths,
		BlockedTagIDs:  t.BlockedTagIDs,
		RequiredTagIDs: t.RequiredTagIDs,
		CanDownload:    t.CanDownload,
		CanViewReels:   t.CanViewReels,
	}
}

// UserSceneData is a user's personal rating and resume position for a scene.
type UserSceneData struct {
	UserID       int
	SceneID      int
	Rating       *int
	ResumeTime   float64
	PlayDuration float64
}

// ReelSegment is a section of the source scene used in a reel, in seconds.
type ReelSegment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// Reel is a short generated vertical clip made from a scene.
type Reel struct {
	ID        int           `json:"id"`
	SceneID   int           `json:"scene_id"`
	Path      string        `json:"path"`
	Duration  float64       `json:"duration"`
	Width     int           `json:"width"`
	Height    int           `json:"height"`
	Source    string        `json:"source"`
	Segments  []ReelSegment `json:"segments"`
	CreatedAt time.Time     `json:"created_at"`
}

// UserReaderWriter stores users, tiers and per-user scene data.
type UserReaderWriter interface {
	Find(ctx context.Context, id int) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindByAPIKey(ctx context.Context, key string) (*User, error)
	All(ctx context.Context) ([]*User, error)
	Count(ctx context.Context) (int, error)
	Create(ctx context.Context, u *User) error
	Update(ctx context.Context, u *User) error
	TouchLogin(ctx context.Context, id int) error
	Destroy(ctx context.Context, id int) error

	FindTier(ctx context.Context, id int) (*UserTier, error)
	AllTiers(ctx context.Context) ([]*UserTier, error)
	CreateTier(ctx context.Context, t *UserTier) error
	UpdateTier(ctx context.Context, t *UserTier) error
	DestroyTier(ctx context.Context, id int) error

	TotalPlayDuration(ctx context.Context, userID int) (float64, error)
}

// ReelReaderWriter stores generated reels.
type ReelReaderWriter interface {
	Create(ctx context.Context, r *Reel) error
	Find(ctx context.Context, id int) (*Reel, error)
	FindBySceneID(ctx context.Context, sceneID int) ([]*Reel, error)
	// Query lists reels newest first, or shuffled when seed is not 0 (the same
	// seed gives the same order, so pages fit together). firstID, when not 0,
	// puts that reel first.
	Query(ctx context.Context, page, perPage, seed, firstID int) ([]*Reel, int, error)
	All(ctx context.Context) ([]*Reel, error)
	Destroy(ctx context.Context, id int) error
}
