package sqlite

// Stasharr: two-step sign in, trusted devices, invites and tier upgrade
// requests.

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// ---------------------------------------------------------------- two-step

// TwoStep is an account's authenticator secret and recovery codes.
type TwoStep struct {
	UserID   int    `db:"user_id"`
	Secret   string `db:"secret"`
	Recovery string `db:"recovery"`
}

// FindTwoStep returns the account's two-step settings, or nil when off.
func FindTwoStep(ctx context.Context, userID int) (*TwoStep, error) {
	var t TwoStep
	err := dbWrapper.Get(ctx, &t, "SELECT user_id, secret, recovery FROM user_two_step WHERE user_id = ?", userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// TwoStepUserIDs lists the accounts with two-step sign in on.
func TwoStepUserIDs(ctx context.Context) (map[int]bool, error) {
	var ids []int
	if err := dbWrapper.Select(ctx, &ids, "SELECT user_id FROM user_two_step"); err != nil {
		return nil, err
	}
	ret := map[int]bool{}
	for _, id := range ids {
		ret[id] = true
	}
	return ret, nil
}

// SaveTwoStep turns two-step sign in on, or saves new recovery codes.
func SaveTwoStep(ctx context.Context, t TwoStep) error {
	_, err := dbWrapper.Exec(ctx, `INSERT INTO user_two_step (user_id, secret, recovery, created_at) VALUES (?, ?, ?, ?)
ON CONFLICT(user_id) DO UPDATE SET secret = excluded.secret, recovery = excluded.recovery`,
		t.UserID, t.Secret, t.Recovery, Timestamp{time.Now()})
	return err
}

// DeleteTwoStep turns two-step sign in off and forgets the account's
// trusted devices.
func DeleteTwoStep(ctx context.Context, userID int) error {
	if _, err := dbWrapper.Exec(ctx, "DELETE FROM user_two_step WHERE user_id = ?", userID); err != nil {
		return err
	}
	return ForgetDevices(ctx, userID)
}

// AddDevice trusts a device for an account.
func AddDevice(ctx context.Context, userID int, tokenHash, name string) error {
	now := Timestamp{time.Now()}
	_, err := dbWrapper.Exec(ctx, "INSERT INTO user_devices (user_id, token_hash, name, created_at, last_used_at) VALUES (?, ?, ?, ?, ?)",
		userID, tokenHash, name, now, now)
	return err
}

// UseDevice reports whether the device is trusted for the account and not
// older than maxAge, and notes that it was used.
func UseDevice(ctx context.Context, userID int, tokenHash string, maxAge time.Duration) (bool, error) {
	var ids []int
	if err := dbWrapper.Select(ctx, &ids, "SELECT id FROM user_devices WHERE user_id = ? AND token_hash = ? AND created_at >= ?",
		userID, tokenHash, Timestamp{time.Now().Add(-maxAge)}); err != nil {
		return false, err
	}
	if len(ids) == 0 {
		return false, nil
	}
	_, err := dbWrapper.Exec(ctx, "UPDATE user_devices SET last_used_at = ? WHERE id = ?", Timestamp{time.Now()}, ids[0])
	return true, err
}

// ForgetDevices removes every trusted device of an account.
func ForgetDevices(ctx context.Context, userID int) error {
	_, err := dbWrapper.Exec(ctx, "DELETE FROM user_devices WHERE user_id = ?", userID)
	return err
}

// CountDevices counts an account's trusted devices.
func CountDevices(ctx context.Context, userID int) (int, error) {
	var n int
	err := dbWrapper.Get(ctx, &n, "SELECT COUNT(*) FROM user_devices WHERE user_id = ?", userID)
	return n, err
}

// ---------------------------------------------------------------- invites

// Invite lets someone make an account with a tier.
type Invite struct {
	ID        int        `db:"id"`
	Code      string     `db:"code"`
	TierID    *int       `db:"tier_id"`
	Role      string     `db:"role"`
	MaxUses   int        `db:"max_uses"`
	Uses      int        `db:"uses"`
	Note      string     `db:"note"`
	ExpiresAt *Timestamp `db:"expires_at"`
	CreatedAt Timestamp  `db:"created_at"`
}

// Usable reports whether the invite can still make an account.
func (i *Invite) Usable() bool {
	if i.Uses >= i.MaxUses {
		return false
	}
	return i.ExpiresAt == nil || time.Now().Before(i.ExpiresAt.Timestamp)
}

const inviteColumns = "id, code, tier_id, role, max_uses, uses, note, expires_at, created_at"

func AllInvites(ctx context.Context) ([]*Invite, error) {
	var ret []*Invite
	err := dbWrapper.Select(ctx, &ret, "SELECT "+inviteColumns+" FROM user_invites ORDER BY id DESC")
	return ret, err
}

func FindInviteByCode(ctx context.Context, code string) (*Invite, error) {
	var i Invite
	err := dbWrapper.Get(ctx, &i, "SELECT "+inviteColumns+" FROM user_invites WHERE code = ?", code)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func CreateInvite(ctx context.Context, i *Invite) error {
	i.CreatedAt = Timestamp{time.Now()}
	res, err := dbWrapper.Exec(ctx, "INSERT INTO user_invites (code, tier_id, role, max_uses, uses, note, expires_at, created_at) VALUES (?, ?, ?, ?, 0, ?, ?, ?)",
		i.Code, i.TierID, i.Role, i.MaxUses, i.Note, i.ExpiresAt, i.CreatedAt)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	i.ID = int(id)
	return err
}

func DeleteInvite(ctx context.Context, id int) error {
	_, err := dbWrapper.Exec(ctx, "DELETE FROM user_invites WHERE id = ?", id)
	return err
}

// UseInvite counts one use; false when the invite was used up meanwhile.
func UseInvite(ctx context.Context, id int) (bool, error) {
	res, err := dbWrapper.Exec(ctx, "UPDATE user_invites SET uses = uses + 1 WHERE id = ? AND uses < max_uses", id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// ---------------------------------------------------------------- tier requests

// TierRequest asks the admin for another tier.
type TierRequest struct {
	ID        int        `db:"id"`
	UserID    int        `db:"user_id"`
	Username  string     `db:"username"`
	TierID    int        `db:"tier_id"`
	TierName  string     `db:"tier_name"`
	Message   string     `db:"message"`
	Status    string     `db:"status"`
	CreatedAt Timestamp  `db:"created_at"`
	DecidedAt *Timestamp `db:"decided_at"`
}

const tierRequestSelect = `SELECT r.id, r.user_id, u.username, r.tier_id, t.name AS tier_name, r.message, r.status, r.created_at, r.decided_at
FROM tier_requests r JOIN users u ON u.id = r.user_id JOIN user_tiers t ON t.id = r.tier_id`

// TierRequests lists requests: pending first, then the newest decided.
func TierRequests(ctx context.Context, userID *int) ([]*TierRequest, error) {
	var ret []*TierRequest
	q := tierRequestSelect
	var args []interface{}
	if userID != nil {
		q += " WHERE r.user_id = ?"
		args = append(args, *userID)
	}
	q += " ORDER BY CASE r.status WHEN 'PENDING' THEN 0 ELSE 1 END, r.id DESC LIMIT 100"
	err := dbWrapper.Select(ctx, &ret, q, args...)
	return ret, err
}

func FindTierRequest(ctx context.Context, id int) (*TierRequest, error) {
	var r TierRequest
	err := dbWrapper.Get(ctx, &r, tierRequestSelect+" WHERE r.id = ?", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// CreateTierRequest replaces the user's pending request with a new one.
func CreateTierRequest(ctx context.Context, userID, tierID int, message string) (int, error) {
	if _, err := dbWrapper.Exec(ctx, "DELETE FROM tier_requests WHERE user_id = ? AND status = 'PENDING'", userID); err != nil {
		return 0, err
	}
	res, err := dbWrapper.Exec(ctx, "INSERT INTO tier_requests (user_id, tier_id, message, status, created_at) VALUES (?, ?, ?, 'PENDING', ?)",
		userID, tierID, message, Timestamp{time.Now()})
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func DecideTierRequest(ctx context.Context, id int, status string) error {
	_, err := dbWrapper.Exec(ctx, "UPDATE tier_requests SET status = ?, decided_at = ? WHERE id = ?", status, Timestamp{time.Now()}, id)
	return err
}

func CancelTierRequest(ctx context.Context, userID int) error {
	_, err := dbWrapper.Exec(ctx, "DELETE FROM tier_requests WHERE user_id = ? AND status = 'PENDING'", userID)
	return err
}

func PendingTierRequests(ctx context.Context) (int, error) {
	var n int
	err := dbWrapper.Get(ctx, &n, "SELECT COUNT(*) FROM tier_requests WHERE status = 'PENDING'")
	return n, err
}
