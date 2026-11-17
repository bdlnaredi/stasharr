package api

// Stasharr: two-step sign in, invites and tier upgrade requests.

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/qr"
	"github.com/stashapp/stash/pkg/sqlite"
)

// signedIn returns the account id for two-step sign in (0 for the owner).
func signedIn(ctx context.Context) (*access.Principal, error) {
	p := access.FromContext(ctx)
	if p == nil {
		return nil, errors.New("sign in first")
	}
	return p, nil
}

func (r *userResolver) TwoStepEnabled(ctx context.Context, obj *models.User) (bool, error) {
	return manager.TwoStepEnabled(ctx, obj.ID), nil
}

func (r *queryResolver) MyTwoStep(ctx context.Context) (*TwoStepStatus, error) {
	p, err := signedIn(ctx)
	if err != nil {
		return nil, err
	}
	ret := &TwoStepStatus{}
	err = r.withReadTxn(ctx, func(ctx context.Context) error {
		t, err := sqlite.FindTwoStep(ctx, p.UserID)
		if err != nil || t == nil {
			return err
		}
		ret.Enabled = true
		var hashes []string
		_ = json.Unmarshal([]byte(t.Recovery), &hashes)
		ret.RecoveryLeft = len(hashes)
		ret.TrustedDevices, err = sqlite.CountDevices(ctx, p.UserID)
		return err
	})
	return ret, err
}

func (r *mutationResolver) TwoStepStart(ctx context.Context) (*TwoStepSetup, error) {
	p, err := signedIn(ctx)
	if err != nil {
		return nil, err
	}
	if manager.TwoStepEnabled(ctx, p.UserID) {
		return nil, errors.New("two-step sign in is already on")
	}
	s := manager.StartTwoStep(p.UserID, p.Username, config.GetInstance().GetSiteName())
	code, err := qr.Encode(s.URL)
	if err != nil {
		return nil, err
	}
	return &TwoStepSetup{Secret: s.Secret, URL: s.URL, QRSVG: code.SVG()}, nil
}

func (r *mutationResolver) TwoStepConfirm(ctx context.Context, code string) ([]string, error) {
	p, err := signedIn(ctx)
	if err != nil {
		return nil, err
	}
	codes, err := manager.ConfirmTwoStep(ctx, p.UserID, code)
	if err == nil {
		logger.Infof("[two-step] %q turned on two-step sign in", p.Username)
	}
	return codes, err
}

// checkPassword checks the signed in account's password.
func (r *Resolver) checkPassword(ctx context.Context, p *access.Principal, password string) error {
	cfg := config.GetInstance()
	if p.IsOwner() {
		if !cfg.ValidateCredentials(cfg.GetUsername(), password) {
			return errors.New("the password is not right")
		}
		return nil
	}
	return r.withReadTxn(ctx, func(ctx context.Context) error {
		u, err := r.repository.User.Find(ctx, p.UserID)
		if err != nil {
			return err
		}
		if u == nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
			return errors.New("the password is not right")
		}
		return nil
	})
}

func (r *mutationResolver) TwoStepDisable(ctx context.Context, password string) (bool, error) {
	p, err := signedIn(ctx)
	if err != nil {
		return false, err
	}
	if err := r.checkPassword(ctx, p, password); err != nil {
		return false, err
	}
	if err := manager.TurnOffTwoStep(ctx, p.UserID); err != nil {
		return false, err
	}
	logger.Infof("[two-step] %q turned off two-step sign in", p.Username)
	return true, nil
}

func (r *mutationResolver) TwoStepRecoveryCodes(ctx context.Context) ([]string, error) {
	p, err := signedIn(ctx)
	if err != nil {
		return nil, err
	}
	return manager.NewRecoveryCodes(ctx, p.UserID)
}

func (r *mutationResolver) TwoStepForgetDevices(ctx context.Context) (bool, error) {
	p, err := signedIn(ctx)
	if err != nil {
		return false, err
	}
	err = r.withTxn(ctx, func(ctx context.Context) error {
		return sqlite.ForgetDevices(ctx, p.UserID)
	})
	return err == nil, err
}

func (r *mutationResolver) UserResetTwoStep(ctx context.Context, id string) (bool, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return false, err
	}
	uid, err := strconv.Atoi(id)
	if err != nil {
		return false, fmt.Errorf("invalid id %q", id)
	}
	if err := manager.TurnOffTwoStep(ctx, uid); err != nil {
		return false, err
	}
	return true, nil
}

// ---------------------------------------------------------------- invites

func (r *Resolver) toInvite(ctx context.Context, i *sqlite.Invite) (*UserInvite, error) {
	ret := &UserInvite{
		ID:        strconv.Itoa(i.ID),
		Code:      i.Code,
		Role:      access.Role(i.Role),
		MaxUses:   i.MaxUses,
		Uses:      i.Uses,
		Note:      i.Note,
		CreatedAt: i.CreatedAt.Timestamp,
		Usable:    i.Usable(),
	}
	if i.ExpiresAt != nil {
		t := i.ExpiresAt.Timestamp
		ret.ExpiresAt = &t
	}
	if i.TierID != nil {
		t, err := r.repository.User.FindTier(ctx, *i.TierID)
		if err != nil {
			return nil, err
		}
		ret.Tier = t
	}
	return ret, nil
}

func (r *queryResolver) UserInvites(ctx context.Context) ([]*UserInvite, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	ret := []*UserInvite{}
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		list, err := sqlite.AllInvites(ctx)
		if err != nil {
			return err
		}
		for _, i := range list {
			inv, err := r.toInvite(ctx, i)
			if err != nil {
				return err
			}
			ret = append(ret, inv)
		}
		return nil
	})
	return ret, err
}

func inviteCode() string {
	b := make([]byte, 15)
	_, _ = rand.Read(b)
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b))
}

func (r *mutationResolver) UserInviteCreate(ctx context.Context, input UserInviteInput) (*UserInvite, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	if err := requireMultiUser(); err != nil {
		return nil, err
	}
	tierID, err := parseOptionalID(input.TierID)
	if err != nil {
		return nil, err
	}
	inv := &sqlite.Invite{Code: inviteCode(), TierID: tierID, Role: string(access.RoleViewer), MaxUses: 1}
	if input.Role != nil {
		switch *input.Role {
		case access.RoleViewer, access.RoleEditor:
			inv.Role = string(*input.Role)
		default:
			return nil, errors.New("invites can make viewers or editors only")
		}
	}
	if input.MaxUses != nil {
		if *input.MaxUses < 1 || *input.MaxUses > 1000 {
			return nil, errors.New("uses must be between 1 and 1000")
		}
		inv.MaxUses = *input.MaxUses
	}
	if input.Days != nil && *input.Days > 0 {
		t := sqlite.Timestamp{Timestamp: time.Now().Add(time.Duration(*input.Days) * 24 * time.Hour)}
		inv.ExpiresAt = &t
	}
	if input.Note != nil {
		inv.Note = strings.TrimSpace(*input.Note)
	}
	var ret *UserInvite
	err = r.withTxn(ctx, func(ctx context.Context) error {
		if err := r.checkTierExists(ctx, tierID); err != nil {
			return err
		}
		if err := sqlite.CreateInvite(ctx, inv); err != nil {
			return err
		}
		ret, err = r.toInvite(ctx, inv)
		return err
	})
	return ret, err
}

func (r *mutationResolver) UserInviteDelete(ctx context.Context, id string) (bool, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return false, err
	}
	iid, err := strconv.Atoi(id)
	if err != nil {
		return false, fmt.Errorf("invalid id %q", id)
	}
	err = r.withTxn(ctx, func(ctx context.Context) error {
		return sqlite.DeleteInvite(ctx, iid)
	})
	return err == nil, err
}

// ---------------------------------------------------------------- tier requests

func (r *Resolver) toTierRequest(ctx context.Context, t *sqlite.TierRequest) (*TierRequest, error) {
	tier, err := r.repository.User.FindTier(ctx, t.TierID)
	if err != nil {
		return nil, err
	}
	if tier == nil {
		tier = &models.UserTier{ID: t.TierID, Name: t.TierName}
	}
	ret := &TierRequest{
		ID:        strconv.Itoa(t.ID),
		Username:  t.Username,
		Tier:      tier,
		Message:   t.Message,
		Status:    TierRequestStatus(t.Status),
		CreatedAt: t.CreatedAt.Timestamp,
	}
	if t.DecidedAt != nil {
		d := t.DecidedAt.Timestamp
		ret.DecidedAt = &d
	}
	return ret, nil
}

func (r *Resolver) tierRequests(ctx context.Context, userID *int) ([]*TierRequest, error) {
	ret := []*TierRequest{}
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		list, err := sqlite.TierRequests(ctx, userID)
		if err != nil {
			return err
		}
		for _, t := range list {
			x, err := r.toTierRequest(ctx, t)
			if err != nil {
				return err
			}
			ret = append(ret, x)
		}
		return nil
	})
	return ret, err
}

func (r *queryResolver) TierChoices(ctx context.Context) ([]*TierChoice, error) {
	p, err := signedIn(ctx)
	if err != nil {
		return nil, err
	}
	ret := []*TierChoice{}
	if p.IsOwner() || p.Role == access.RoleAdmin {
		return ret, nil
	}
	var current *int
	err = r.withReadTxn(ctx, func(ctx context.Context) error {
		u, err := r.repository.User.Find(ctx, p.UserID)
		if err != nil {
			return err
		}
		if u != nil {
			current = u.TierID
		}
		tiers, err := r.repository.User.AllTiers(ctx)
		if err != nil {
			return err
		}
		for _, t := range tiers {
			ret = append(ret, &TierChoice{
				ID:          strconv.Itoa(t.ID),
				Name:        t.Name,
				Description: description(t.Description),
				Current:     current != nil && *current == t.ID,
			})
		}
		return nil
	})
	return ret, err
}

func (r *queryResolver) MyTierRequests(ctx context.Context) ([]*TierRequest, error) {
	p, err := signedIn(ctx)
	if err != nil {
		return nil, err
	}
	if p.IsOwner() {
		return []*TierRequest{}, nil
	}
	return r.tierRequests(ctx, &p.UserID)
}

func (r *queryResolver) TierRequests(ctx context.Context) ([]*TierRequest, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	return r.tierRequests(ctx, nil)
}

func (r *mutationResolver) TierRequestCreate(ctx context.Context, tierID string, message *string) (*TierRequest, error) {
	p, err := signedIn(ctx)
	if err != nil {
		return nil, err
	}
	if p.IsOwner() {
		return nil, errors.New("the owner account has every tier")
	}
	tid, err := strconv.Atoi(tierID)
	if err != nil {
		return nil, fmt.Errorf("invalid id %q", tierID)
	}
	msg := ""
	if message != nil {
		msg = strings.TrimSpace(*message)
		if len(msg) > 1000 {
			msg = msg[:1000]
		}
	}
	var ret *TierRequest
	tierName := ""
	err = r.withTxn(ctx, func(ctx context.Context) error {
		t, err := r.repository.User.FindTier(ctx, tid)
		if err != nil {
			return err
		}
		if t == nil {
			return errors.New("that tier does not exist")
		}
		tierName = t.Name
		id, err := sqlite.CreateTierRequest(ctx, p.UserID, tid, msg)
		if err != nil {
			return err
		}
		req, err := sqlite.FindTierRequest(ctx, id)
		if err != nil {
			return err
		}
		ret, err = r.toTierRequest(ctx, req)
		return err
	})
	if err == nil {
		text := fmt.Sprintf("%s asks for the %s tier.", p.Username, tierName)
		if msg != "" {
			text += " “" + msg + "”"
		}
		manager.Notify(manager.EventAccounts, "Tier upgrade request", text)
	}
	return ret, err
}

func (r *mutationResolver) TierRequestCancel(ctx context.Context) (bool, error) {
	p, err := signedIn(ctx)
	if err != nil {
		return false, err
	}
	err = r.withTxn(ctx, func(ctx context.Context) error {
		return sqlite.CancelTierRequest(ctx, p.UserID)
	})
	return err == nil, err
}

func (r *mutationResolver) TierRequestDecide(ctx context.Context, id string, approve bool) (*TierRequest, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	rid, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id %q", id)
	}
	var ret *TierRequest
	err = r.withTxn(ctx, func(ctx context.Context) error {
		req, err := sqlite.FindTierRequest(ctx, rid)
		if err != nil {
			return err
		}
		if req == nil {
			return errors.New("request not found")
		}
		if req.Status != "PENDING" {
			return errors.New("this request was already answered")
		}
		status := "DECLINED"
		if approve {
			status = "APPROVED"
			u, err := r.repository.User.Find(ctx, req.UserID)
			if err != nil {
				return err
			}
			if u == nil {
				return errors.New("the account no longer exists")
			}
			tid := req.TierID
			u.TierID = &tid
			if err := r.repository.User.Update(ctx, u); err != nil {
				return err
			}
		}
		if err := sqlite.DecideTierRequest(ctx, rid, status); err != nil {
			return err
		}
		req, err = sqlite.FindTierRequest(ctx, rid)
		if err != nil {
			return err
		}
		ret, err = r.toTierRequest(ctx, req)
		return err
	})
	if err == nil && approve {
		userService().Invalidate()
	}
	return ret, err
}

func description(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
