package session

import "context"

type SessionConfig interface {
	GetUsername() string
	GetAPIKey() string

	GetSessionStoreKey() []byte
	GetMaxSessionAge() int
	ValidateCredentials(username string, password string) bool
}

// UserDirectory authenticates additional database users. It is optional:
// without it only the owner account configured in settings can log in.
type UserDirectory interface {
	// ValidateUser reports whether username/password belong to an enabled user.
	ValidateUser(ctx context.Context, username string, password string) bool
	// UsernameForAPIKey returns the user owning an API key.
	UsernameForAPIKey(ctx context.Context, apiKey string) (string, bool)
}
