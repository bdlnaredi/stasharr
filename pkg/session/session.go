// Package session provides session authentication and management for the application.
package session

import (
	"context"
	"errors"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/stashapp/stash/pkg/logger"
)

type key int

const (
	contextUser key = iota
	contextVisitedPlugins
	contextLocalRequest
)

const (
	userIDKey             = "userID"
	visitedPluginHooksKey = "visitedPluginsHooks"
)

const (
	ApiKeyHeader    = "ApiKey"
	ApiKeyParameter = "apikey"
)

const (
	cookieName      = "session"
	usernameFormKey = "username"
	passwordFormKey = "password"
)

type InvalidCredentialsError struct {
	Username string
}

func (e InvalidCredentialsError) Error() string {
	// don't leak the username
	return "invalid credentials"
}

var ErrUnauthorized = errors.New("unauthorized")

// Stasharr: two-step sign in. ErrTwoStepNeeded means the password was right
// and the account needs the code from its authenticator app;
// ErrTwoStepWrong means the code was not right.
var (
	ErrTwoStepNeeded = errors.New("two-step code needed")
	ErrTwoStepWrong  = errors.New("two-step code is not right")
)

// TwoStepChecker checks the second step after a right password. It returns
// a new device token to remember when a code was used.
type TwoStepChecker interface {
	SecondStep(ctx context.Context, username, code, device, deviceName string) (string, error)
}

const (
	codeFormKey   = "code"
	deviceFormKey = "device"
	// DeviceCookie keeps the trusted device token in browsers
	DeviceCookie = "stasharr_device"
	// DeviceHeader sends a new trusted device token to apps
	DeviceHeader = "X-Stasharr-Device"
)

type Store struct {
	sessionStore *sessions.CookieStore
	config       SessionConfig
	users        UserDirectory
}

// SetUserDirectory enables login for database users.
func (s *Store) SetUserDirectory(d UserDirectory) {
	s.users = d
}

func NewStore(c SessionConfig) *Store {
	ret := &Store{
		sessionStore: sessions.NewCookieStore(c.GetSessionStoreKey()),
		config:       c,
	}

	ret.sessionStore.MaxAge(c.GetMaxSessionAge())
	ret.sessionStore.Options.SameSite = http.SameSiteLaxMode

	return ret
}

func (s *Store) Login(w http.ResponseWriter, r *http.Request) error {
	// ignore error - we want a new session regardless
	newSession, _ := s.sessionStore.Get(r, cookieName)

	username := r.FormValue(usernameFormKey)
	password := r.FormValue(passwordFormKey)

	// authenticate the user: first the owner account, then database users
	ownerName := s.config.GetUsername()
	validOwner := (ownerName == "" || username == ownerName) && s.config.ValidateCredentials(username, password)
	if !validOwner {
		if s.users == nil || !s.users.ValidateUser(r.Context(), username, password) {
			logger.Warnf("Failed login attempt from %s", r.RemoteAddr)
			return &InvalidCredentialsError{Username: username}
		}
	}

	// the second step, when the account has one
	if ts, ok := s.users.(TwoStepChecker); ok && s.users != nil {
		device := r.FormValue(deviceFormKey)
		if device == "" {
			if c, err := r.Cookie(DeviceCookie); err == nil {
				device = c.Value
			}
		}
		newDevice, err := ts.SecondStep(r.Context(), username, r.FormValue(codeFormKey), device, r.UserAgent())
		if err != nil {
			return err
		}
		if newDevice != "" {
			http.SetCookie(w, &http.Cookie{
				Name:     DeviceCookie,
				Value:    newDevice,
				Path:     "/",
				MaxAge:   90 * 24 * 3600,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
			w.Header().Set(DeviceHeader, newDevice)
		}
	}

	logger.Infof("User %q logged in", username)

	newSession.Values[userIDKey] = username

	err := newSession.Save(r, w)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) Logout(w http.ResponseWriter, r *http.Request) error {
	session, err := s.sessionStore.Get(r, cookieName)
	if err != nil {
		return err
	}

	delete(session.Values, userIDKey)
	session.Options.MaxAge = -1

	err = session.Save(r, w)
	if err != nil {
		return err
	}

	logger.Infof("User logged out")

	return nil
}

func (s *Store) GetSessionUserID(w http.ResponseWriter, r *http.Request) (string, error) {
	session, err := s.sessionStore.Get(r, cookieName)
	// ignore errors and treat as an empty user id, so that we handle expired
	// cookie
	if err != nil {
		return "", nil
	}

	if !session.IsNew {
		val := session.Values[userIDKey]

		// refresh the cookie
		err = session.Save(r, w)
		if err != nil {
			return "", err
		}

		ret, _ := val.(string)

		return ret, nil
	}

	return "", nil
}

func SetCurrentUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, contextUser, userID)
}

// GetCurrentUserID gets the current user id from the provided context
func GetCurrentUserID(ctx context.Context) *string {
	userCtxVal := ctx.Value(contextUser)
	if userCtxVal != nil {
		currentUser := userCtxVal.(string)
		return &currentUser
	}

	return nil
}

func (s *Store) Authenticate(w http.ResponseWriter, r *http.Request) (userID string, err error) {
	c := s.config

	// translate api key into current user, if present
	apiKey := r.Header.Get(ApiKeyHeader)

	// try getting the api key as a query parameter
	if apiKey == "" {
		apiKey = r.URL.Query().Get(ApiKeyParameter)
	}

	if apiKey != "" {
		// match against configured API and set userID to the
		// configured username. In future, we'll want to
		// get the username from the key.
		if ownerKey := c.GetAPIKey(); ownerKey != "" && ownerKey == apiKey {
			userID = c.GetUsername()
		} else if s.users != nil {
			name, ok := s.users.UsernameForAPIKey(r.Context(), apiKey)
			if !ok {
				return "", ErrUnauthorized
			}
			userID = name
		} else {
			return "", ErrUnauthorized
		}
	} else {
		// handle session
		userID, err = s.GetSessionUserID(w, r)
	}

	if err != nil {
		return "", err
	}

	return
}

// SwitchUser moves the current browser session to another account, for
// Stasharr profiles. The caller checks that the switch is allowed.
func (s *Store) SwitchUser(w http.ResponseWriter, r *http.Request, username string) error {
	session, _ := s.sessionStore.Get(r, cookieName)
	session.Values[userIDKey] = username
	logger.Infof("Session moved to profile %q", username)
	return session.Save(r, w)
}
