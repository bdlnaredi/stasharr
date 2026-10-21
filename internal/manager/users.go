package manager

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
)

// UserService resolves logins, API keys and request principals for the
// additional database users of this fork.
type UserService struct {
	Repository models.Repository
	Config     *config.Config

	mu    sync.Mutex
	cache map[string]cachedPrincipal
}

type cachedPrincipal struct {
	p       *access.Principal
	expires time.Time
}

const principalCacheTTL = 30 * time.Second

var ErrUserNotFound = errors.New("user not found or disabled")

func NewUserService(repo models.Repository, cfg *config.Config) *UserService {
	return &UserService{Repository: repo, Config: cfg, cache: map[string]cachedPrincipal{}}
}

// HashPassword hashes a password for storage.
func HashPassword(password string) (string, error) {
	if len(password) < 6 {
		return "", errors.New("password must be at least 6 characters")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// GenerateAPIKey returns a random API key.
func GenerateUserAPIKey() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return "stu_" + hex.EncodeToString(b)
}

// Invalidate clears the principal cache, after users or tiers change.
func (s *UserService) Invalidate() {
	s.mu.Lock()
	s.cache = map[string]cachedPrincipal{}
	s.mu.Unlock()
}

func (s *UserService) dbReady() bool {
	return s.Repository.User != nil && s.Repository.TxnManager != nil
}

// IsOwner reports whether username is the owner account.
func (s *UserService) IsOwner(username string) bool {
	owner := s.Config.GetUsername()
	return owner != "" && strings.EqualFold(owner, username)
}

func (s *UserService) findUser(ctx context.Context, username string) (*models.User, error) {
	if !s.dbReady() {
		return nil, nil
	}
	var u *models.User
	err := s.Repository.WithReadTxn(ctx, func(ctx context.Context) error {
		var err error
		u, err = s.Repository.User.FindByUsername(ctx, username)
		return err
	})
	return u, err
}

// ValidateUser implements session.UserDirectory.
func (s *UserService) ValidateUser(ctx context.Context, username string, password string) bool {
	// users can only exist alongside an owner account
	if !s.Config.HasCredentials() || s.IsOwner(username) {
		return false
	}
	ctx = context.WithoutCancel(ctx)
	u, err := s.findUser(ctx, username)
	if err != nil {
		logger.Errorf("looking up user: %v", err)
		return false
	}
	if u == nil || u.Disabled {
		// spend similar time as a real check
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$7EqJtq98hPqEX7fNZaFWoO5nYz8b6.bmFHnB3V5CQKQ1jYvYJXy2a"), []byte(password))
		return false
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return false
	}
	if err := s.Repository.WithTxn(ctx, func(ctx context.Context) error {
		return s.Repository.User.TouchLogin(ctx, u.ID)
	}); err != nil {
		logger.Warnf("updating last login: %v", err)
	}
	return true
}

// UsernameForAPIKey implements session.UserDirectory.
func (s *UserService) UsernameForAPIKey(ctx context.Context, apiKey string) (string, bool) {
	if !s.dbReady() || !s.Config.HasCredentials() {
		return "", false
	}
	var u *models.User
	if err := s.Repository.WithReadTxn(context.WithoutCancel(ctx), func(ctx context.Context) error {
		var err error
		u, err = s.Repository.User.FindByAPIKey(ctx, apiKey)
		return err
	}); err != nil {
		logger.Errorf("looking up api key: %v", err)
		return "", false
	}
	if u == nil || u.Disabled {
		return "", false
	}
	return u.Username, true
}

// Principal returns the principal for an authenticated username.
// It returns ErrUserNotFound when a database user was removed or disabled.
func (s *UserService) Principal(ctx context.Context, username string) (*access.Principal, error) {
	if s.IsOwner(username) {
		return access.OwnerPrincipal(s.Config.GetUsername()), nil
	}

	key := strings.ToLower(username)
	s.mu.Lock()
	if c, ok := s.cache[key]; ok && time.Now().Before(c.expires) {
		s.mu.Unlock()
		return c.p, nil
	}
	s.mu.Unlock()

	var p *access.Principal
	err := s.Repository.WithReadTxn(context.WithoutCancel(ctx), func(ctx context.Context) error {
		u, err := s.Repository.User.FindByUsername(ctx, username)
		if err != nil {
			return err
		}
		if u == nil || u.Disabled {
			return ErrUserNotFound
		}
		p = &access.Principal{
			UserID:   u.ID,
			Username: u.Username,
			Role:     u.Role,
			TierID:   u.TierID,
			Policy:   access.Policy{CanDownload: true, CanViewReels: true},
		}
		if u.TierID != nil {
			t, err := s.Repository.User.FindTier(ctx, *u.TierID)
			if err != nil {
				return err
			}
			if t != nil {
				p.TierName = t.Name
				p.Policy = t.Policy()
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.cache[key] = cachedPrincipal{p: p, expires: time.Now().Add(principalCacheTTL)}
	s.mu.Unlock()
	return p, nil
}

// ValidateUsername checks a new username.
func (s *UserService) ValidateUsername(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("username is required")
	}
	if len(name) > 64 {
		return errors.New("username is too long")
	}
	if s.IsOwner(name) {
		return fmt.Errorf("%q is the owner account name", name)
	}
	return nil
}
