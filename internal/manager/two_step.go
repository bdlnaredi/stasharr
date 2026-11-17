package manager

// Two-step sign in with an authenticator app (TOTP, RFC 6238), recovery
// codes, trusted devices, and a limit on wrong passwords.

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/session"
	"github.com/stashapp/stash/pkg/sqlite"
)

// TrustedDeviceAge is how long a device that passed two-step sign in is
// trusted.
const TrustedDeviceAge = 90 * 24 * time.Hour

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

func totpCode(secret []byte, step int64) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(step))
	mac := hmac.New(sha1.New, secret)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	v := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", v%1000000)
}

// lastSteps remembers the last accepted step per account, so a code works
// only once.
var lastSteps sync.Map

// checkTOTP accepts the code for now, 30 seconds before or after.
func checkTOTP(userID int, secretB32, code string) bool {
	secret, err := b32.DecodeString(strings.ToUpper(secretB32))
	if err != nil {
		return false
	}
	code = strings.ReplaceAll(strings.TrimSpace(code), " ", "")
	if len(code) != 6 {
		return false
	}
	now := time.Now().Unix() / 30
	for d := int64(-1); d <= 1; d++ {
		step := now + d
		if hmac.Equal([]byte(totpCode(secret, step)), []byte(code)) {
			if prev, ok := lastSteps.Load(userID); ok && prev.(int64) >= step {
				return false
			}
			lastSteps.Store(userID, step)
			return true
		}
	}
	return false
}

func hashToken(s string) string {
	sum := sha256.Sum256([]byte(strings.ToUpper(strings.TrimSpace(s))))
	return hex.EncodeToString(sum[:])
}

func randomString(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b32.EncodeToString(b)
}

// ---------------------------------------------------------------- setup

type pendingSetup struct {
	secret  string
	expires time.Time
}

var pendingSetups sync.Map

// TwoStepSetup is what the account needs to add Stasharr to an
// authenticator app.
type TwoStepSetup struct {
	Secret string
	URL    string
}

// StartTwoStep makes a new secret for an account; it is used once the
// account confirms a code from the app.
func StartTwoStep(userID int, username, siteName string) TwoStepSetup {
	secret := randomString(20)
	pendingSetups.Store(userID, pendingSetup{secret: secret, expires: time.Now().Add(15 * time.Minute)})
	if siteName == "" {
		siteName = "Stasharr"
	}
	label := url.PathEscape(siteName + ":" + username)
	u := fmt.Sprintf("otpauth://totp/%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30", label, secret, url.QueryEscape(siteName))
	return TwoStepSetup{Secret: secret, URL: u}
}

// ConfirmTwoStep turns two-step sign in on when the code matches the new
// secret, and returns the recovery codes.
func ConfirmTwoStep(ctx context.Context, userID int, code string) ([]string, error) {
	v, ok := pendingSetups.Load(userID)
	if !ok || time.Now().After(v.(pendingSetup).expires) {
		return nil, errors.New("the setup expired; start again")
	}
	secret := v.(pendingSetup).secret
	if !checkTOTP(userID, secret, code) {
		return nil, errors.New("that code is not right; check the time on your phone and try the newest code")
	}
	codes, hashes := makeRecoveryCodes()
	b, err := json.Marshal(hashes)
	if err != nil {
		return nil, err
	}
	r := GetInstance().Repository
	if err := r.WithTxn(ctx, func(ctx context.Context) error {
		if err := sqlite.SaveTwoStep(ctx, sqlite.TwoStep{UserID: userID, Secret: secret, Recovery: string(b)}); err != nil {
			return err
		}
		// devices trusted before need the code once
		return sqlite.ForgetDevices(ctx, userID)
	}); err != nil {
		return nil, err
	}
	pendingSetups.Delete(userID)
	return codes, nil
}

func makeRecoveryCodes() (codes, hashes []string) {
	for i := 0; i < 8; i++ {
		raw := strings.ToUpper(randomString(5))[:8]
		c := raw[:4] + "-" + raw[4:]
		codes = append(codes, c)
		hashes = append(hashes, hashToken(strings.ReplaceAll(c, "-", "")))
	}
	return codes, hashes
}

// NewRecoveryCodes replaces an account's recovery codes.
func NewRecoveryCodes(ctx context.Context, userID int) ([]string, error) {
	r := GetInstance().Repository
	var codes []string
	err := r.WithTxn(ctx, func(ctx context.Context) error {
		t, err := sqlite.FindTwoStep(ctx, userID)
		if err != nil {
			return err
		}
		if t == nil {
			return errors.New("two-step sign in is off")
		}
		var hashes []string
		codes, hashes = makeRecoveryCodes()
		b, err := json.Marshal(hashes)
		if err != nil {
			return err
		}
		t.Recovery = string(b)
		return sqlite.SaveTwoStep(ctx, *t)
	})
	return codes, err
}

// TwoStepEnabled reports whether an account has two-step sign in on.
func TwoStepEnabled(ctx context.Context, userID int) bool {
	r := GetInstance().Repository
	on := false
	_ = r.WithReadTxn(context.WithoutCancel(ctx), func(ctx context.Context) error {
		t, err := sqlite.FindTwoStep(ctx, userID)
		on = err == nil && t != nil
		return err
	})
	return on
}

// TurnOffTwoStep turns two-step sign in off for an account.
func TurnOffTwoStep(ctx context.Context, userID int) error {
	r := GetInstance().Repository
	return r.WithTxn(ctx, func(ctx context.Context) error {
		return sqlite.DeleteTwoStep(ctx, userID)
	})
}

// ---------------------------------------------------------------- at sign in

// SecondStep checks the second step for an account whose password was
// right. It returns a new device token to remember when a code was used.
func SecondStep(ctx context.Context, userID int, code, device, deviceName string) (newDevice string, err error) {
	ctx = context.WithoutCancel(ctx)
	r := GetInstance().Repository
	var t *sqlite.TwoStep
	if err := r.WithReadTxn(ctx, func(ctx context.Context) error {
		var err error
		t, err = sqlite.FindTwoStep(ctx, userID)
		return err
	}); err != nil {
		return "", err
	}
	if t == nil {
		return "", nil
	}
	if device != "" {
		ok := false
		if err := r.WithTxn(ctx, func(ctx context.Context) error {
			var err error
			ok, err = sqlite.UseDevice(ctx, userID, hashToken(device), TrustedDeviceAge)
			return err
		}); err != nil {
			return "", err
		}
		if ok {
			return "", nil
		}
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return "", session.ErrTwoStepNeeded
	}
	if !checkTOTP(userID, t.Secret, code) && !useRecoveryCode(ctx, t, code) {
		return "", session.ErrTwoStepWrong
	}
	token := randomString(32)
	if len(deviceName) > 200 {
		deviceName = deviceName[:200]
	}
	if err := r.WithTxn(ctx, func(ctx context.Context) error {
		return sqlite.AddDevice(ctx, userID, hashToken(token), deviceName)
	}); err != nil {
		return "", err
	}
	return token, nil
}

func useRecoveryCode(ctx context.Context, t *sqlite.TwoStep, code string) bool {
	h := hashToken(strings.ReplaceAll(code, "-", ""))
	var hashes []string
	if err := json.Unmarshal([]byte(t.Recovery), &hashes); err != nil {
		return false
	}
	for i, x := range hashes {
		if hmac.Equal([]byte(x), []byte(h)) {
			hashes = append(hashes[:i], hashes[i+1:]...)
			b, err := json.Marshal(hashes)
			if err != nil {
				return false
			}
			t.Recovery = string(b)
			r := GetInstance().Repository
			if err := r.WithTxn(ctx, func(ctx context.Context) error {
				return sqlite.SaveTwoStep(ctx, *t)
			}); err != nil {
				logger.Warnf("[two-step] saving recovery codes: %v", err)
				return false
			}
			logger.Infof("[two-step] a recovery code was used; %d left", len(hashes))
			return true
		}
	}
	return false
}

// SecondStep implements session.TwoStepChecker.
func (s *UserService) SecondStep(ctx context.Context, username, code, device, deviceName string) (string, error) {
	if !s.dbReady() {
		return "", nil
	}
	// before the database is migrated after an update, the two-step tables
	// may not exist yet; nobody can have two-step sign in on then, and the
	// admin must be able to sign in to migrate
	if db := GetInstance().Database; db == nil || db.Ready() != nil || db.NeedsMigration() {
		return "", nil
	}
	id := s.UserIDForLogin(ctx, username)
	if id < 0 {
		return "", nil
	}
	return SecondStep(ctx, id, code, device, deviceName)
}

// UserIDForLogin returns the account id used for two-step sign in: 0 for
// the owner, the user's id otherwise, or -1 when unknown.
func (s *UserService) UserIDForLogin(ctx context.Context, username string) int {
	if s.IsOwner(username) || (!s.Config.HasCredentials()) {
		return 0
	}
	u, err := s.findUser(context.WithoutCancel(ctx), username)
	if err != nil || u == nil {
		return -1
	}
	return u.ID
}

// ---------------------------------------------------------------- wrong passwords

// LoginGuard limits wrong passwords and codes: after 5 for one account
// from one address, or 20 from one address, within 15 minutes, sign in
// from that address is refused for 15 minutes.
type LoginGuard struct {
	mu       sync.Mutex
	failures map[string][]time.Time
}

const (
	loginWindow      = 15 * time.Minute
	loginPerAccount  = 5
	loginPerAddress  = 20
	loginGuardPrefix = "addr:"
)

var Guard = &LoginGuard{failures: map[string][]time.Time{}}

func (g *LoginGuard) recent(key string, now time.Time) []time.Time {
	list := g.failures[key]
	keep := list[:0]
	for _, t := range list {
		if now.Sub(t) < loginWindow {
			keep = append(keep, t)
		}
	}
	if len(keep) == 0 {
		delete(g.failures, key)
		return nil
	}
	g.failures[key] = keep
	return keep
}

// Blocked returns how long sign in is refused, or 0.
func (g *LoginGuard) Blocked(addr, username string) time.Duration {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	for key, limit := range map[string]int{
		addr + "|" + strings.ToLower(username): loginPerAccount,
		loginGuardPrefix + addr:                loginPerAddress,
	} {
		list := g.recent(key, now)
		if len(list) >= limit {
			wait := loginWindow - now.Sub(list[len(list)-limit])
			if wait > 0 {
				return wait
			}
		}
	}
	return 0
}

// Failed records a wrong password or code.
func (g *LoginGuard) Failed(addr, username string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	for _, key := range []string{addr + "|" + strings.ToLower(username), loginGuardPrefix + addr} {
		g.failures[key] = append(g.recent(key, now), now)
	}
	// keep memory small
	if len(g.failures) > 10000 {
		g.failures = map[string][]time.Time{}
	}
}

// Succeeded forgets the account's wrong passwords from that address.
func (g *LoginGuard) Succeeded(addr, username string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.failures, addr+"|"+strings.ToLower(username))
}
