package api

import (
	"context"
	"net/url"
	"time"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/signedurl"
)

// userSigningKey returns the HMAC signing key for a given user.
func userSigningKey(c *config.Config, _ string) []byte {
	return c.GetJWTSignKey()
}

// signedParams generates signed URL query parameters for the given path prefix and user.
func signedParams(c *config.Config, userID string, prefix string) url.Values {
	secret := userSigningKey(c, userID)
	cid := signedurl.GenerateCredentialID(secret, userID)
	expires := time.Now().Add(c.GetSignedURLExpiry())
	return signedurl.SignPrefix(prefix, secret, cid, expires)
}

// resolveCredentialID maps a credential ID back to a username and their signing key.
func resolveCredentialID(c *config.Config, cid string) (string, []byte, bool) {
	username := c.GetUsername()
	secret := userSigningKey(c, username)
	if signedurl.GenerateCredentialID(secret, username) == cid {
		return username, secret, true
	}

	// additional database users
	repo := manager.GetInstance().Repository
	if repo.User == nil {
		return "", nil, false
	}
	var users []*models.User
	if err := repo.WithReadTxn(context.Background(), func(ctx context.Context) error {
		var err error
		users, err = repo.User.All(ctx)
		return err
	}); err != nil {
		return "", nil, false
	}
	for _, u := range users {
		if u.Disabled {
			continue
		}
		secret := userSigningKey(c, u.Username)
		if signedurl.GenerateCredentialID(secret, u.Username) == cid {
			return u.Username, secret, true
		}
	}
	return "", nil, false
}
