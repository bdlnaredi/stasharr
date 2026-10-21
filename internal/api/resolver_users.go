package api

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
)

func (r *Resolver) User() UserResolver         { return &userResolver{r} }
func (r *Resolver) UserTier() UserTierResolver { return &userTierResolver{r} }

type userResolver struct{ *Resolver }
type userTierResolver struct{ *Resolver }

func userService() *manager.UserService {
	return manager.GetInstance().UserService
}

func invalidateUsers() {
	if us := userService(); us != nil {
		us.Invalidate()
	}
}

// --- queries

func (r *queryResolver) Me(ctx context.Context) (*CurrentUser, error) {
	cfg := manager.GetInstance().Config
	p := access.FromContext(ctx)
	if p == nil {
		if !cfg.HasCredentials() {
			// no login configured: single-user mode with full access
			return &CurrentUser{
				Username:         "",
				Role:             access.RoleAdmin,
				IsOwner:          true,
				CanDownload:      true,
				CanViewReels:     true,
				MultiUserEnabled: false,
			}, nil
		}
		return nil, nil
	}

	ret := &CurrentUser{
		Username:         p.Username,
		Role:             p.Role,
		IsOwner:          p.IsOwner(),
		CanDownload:      p.CanDownload(),
		CanViewReels:     p.CanViewReels(),
		MultiUserEnabled: cfg.HasCredentials(),
	}
	if p.TierName != "" {
		t := p.TierName
		ret.Tier = &t
	}
	return ret, nil
}

func (r *queryResolver) Users(ctx context.Context) (ret []*models.User, err error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	err = r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.User.All(ctx)
		return err
	})
	return ret, err
}

func (r *queryResolver) UserTiers(ctx context.Context) (ret []*models.UserTier, err error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	err = r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.User.AllTiers(ctx)
		return err
	})
	return ret, err
}

// --- field resolvers

func (r *userResolver) Tier(ctx context.Context, obj *models.User) (ret *models.UserTier, err error) {
	if obj.TierID == nil {
		return nil, nil
	}
	err = r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.User.FindTier(ctx, *obj.TierID)
		return err
	})
	return ret, err
}

func (r *userResolver) HasAPIKey(ctx context.Context, obj *models.User) (bool, error) {
	return obj.APIKey != nil && *obj.APIKey != "", nil
}

func (r *userTierResolver) tags(ctx context.Context, ids []int) (ret []*models.Tag, err error) {
	if len(ids) == 0 {
		return []*models.Tag{}, nil
	}
	err = r.withReadTxn(ctx, func(ctx context.Context) error {
		// tiers are only visible to admins, who are never restricted
		ret, err = r.repository.Tag.FindMany(ctx, ids)
		return err
	})
	return ret, err
}

func (r *userTierResolver) BlockedTags(ctx context.Context, obj *models.UserTier) ([]*models.Tag, error) {
	return r.tags(ctx, obj.BlockedTagIDs)
}

func (r *userTierResolver) RequiredTags(ctx context.Context, obj *models.UserTier) ([]*models.Tag, error) {
	return r.tags(ctx, obj.RequiredTagIDs)
}

func (r *userTierResolver) UserCount(ctx context.Context, obj *models.UserTier) (int, error) {
	var users []*models.User
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		var err error
		users, err = r.repository.User.All(ctx)
		return err
	}); err != nil {
		return 0, err
	}
	n := 0
	for _, u := range users {
		if u.TierID != nil && *u.TierID == obj.ID {
			n++
		}
	}
	return n, nil
}

// --- mutations

func requireMultiUser() error {
	if !manager.GetInstance().Config.HasCredentials() {
		return errors.New("set an owner username and password in Settings > Security before adding users")
	}
	return nil
}

func parseOptionalID(s *string) (*int, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	id, err := strconv.Atoi(*s)
	if err != nil {
		return nil, fmt.Errorf("invalid id %q", *s)
	}
	return &id, nil
}

func (r *mutationResolver) checkTierExists(ctx context.Context, id *int) error {
	if id == nil {
		return nil
	}
	t, err := r.repository.User.FindTier(ctx, *id)
	if err != nil {
		return err
	}
	if t == nil {
		return fmt.Errorf("tier %d not found", *id)
	}
	return nil
}

func (r *mutationResolver) UserCreate(ctx context.Context, input UserCreateInput) (*models.User, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	if err := requireMultiUser(); err != nil {
		return nil, err
	}
	us := userService()
	username := strings.TrimSpace(input.Username)
	if err := us.ValidateUsername(username); err != nil {
		return nil, err
	}
	if !input.Role.IsValid() {
		return nil, fmt.Errorf("invalid role %q", input.Role)
	}
	hash, err := manager.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}
	tierID, err := parseOptionalID(input.TierID)
	if err != nil {
		return nil, err
	}

	u := &models.User{
		Username:     username,
		PasswordHash: hash,
		Role:         input.Role,
		TierID:       tierID,
		Disabled:     input.Disabled != nil && *input.Disabled,
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		existing, err := r.repository.User.FindByUsername(ctx, username)
		if err != nil {
			return err
		}
		if existing != nil {
			return fmt.Errorf("a user called %q already exists", username)
		}
		if err := r.checkTierExists(ctx, tierID); err != nil {
			return err
		}
		return r.repository.User.Create(ctx, u)
	}); err != nil {
		return nil, err
	}

	invalidateUsers()
	return u, nil
}

func (r *mutationResolver) UserUpdate(ctx context.Context, input UserUpdateInput) (*models.User, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	id, err := strconv.Atoi(input.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid id %q", input.ID)
	}
	translator := changesetTranslator{inputMap: getUpdateInputMap(ctx)}

	var u *models.User
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		u, err = r.repository.User.Find(ctx, id)
		if err != nil {
			return err
		}
		if u == nil {
			return fmt.Errorf("user %d not found", id)
		}

		if input.Username != nil && *input.Username != u.Username {
			name := strings.TrimSpace(*input.Username)
			if err := userService().ValidateUsername(name); err != nil {
				return err
			}
			existing, err := r.repository.User.FindByUsername(ctx, name)
			if err != nil {
				return err
			}
			if existing != nil && existing.ID != u.ID {
				return fmt.Errorf("a user called %q already exists", name)
			}
			u.Username = name
		}
		if input.Password != nil && *input.Password != "" {
			hash, err := manager.HashPassword(*input.Password)
			if err != nil {
				return err
			}
			u.PasswordHash = hash
		}
		if input.Role != nil {
			if !input.Role.IsValid() {
				return fmt.Errorf("invalid role %q", *input.Role)
			}
			u.Role = *input.Role
		}
		if translator.hasField("tier_id") {
			tierID, err := parseOptionalID(input.TierID)
			if err != nil {
				return err
			}
			if err := r.checkTierExists(ctx, tierID); err != nil {
				return err
			}
			u.TierID = tierID
		}
		if input.Disabled != nil {
			u.Disabled = *input.Disabled
		}
		return r.repository.User.Update(ctx, u)
	}); err != nil {
		return nil, err
	}

	invalidateUsers()
	return u, nil
}

func (r *mutationResolver) UserDestroy(ctx context.Context, id string) (bool, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return false, err
	}
	uid, err := strconv.Atoi(id)
	if err != nil {
		return false, fmt.Errorf("invalid id %q", id)
	}
	if p := access.FromContext(ctx); p != nil && p.UserID == uid {
		return false, errors.New("you cannot delete your own account")
	}
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		return r.repository.User.Destroy(ctx, uid)
	}); err != nil {
		return false, err
	}
	invalidateUsers()
	return true, nil
}

func (r *mutationResolver) UserGenerateAPIKey(ctx context.Context, id string, clear *bool) (*string, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	uid, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id %q", id)
	}
	var key *string
	if clear == nil || !*clear {
		k := manager.GenerateUserAPIKey()
		key = &k
	}
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		u, err := r.repository.User.Find(ctx, uid)
		if err != nil {
			return err
		}
		if u == nil {
			return fmt.Errorf("user %d not found", uid)
		}
		u.APIKey = key
		return r.repository.User.Update(ctx, u)
	}); err != nil {
		return nil, err
	}
	invalidateUsers()
	return key, nil
}

func tagIDs(ids []string) ([]int, error) {
	if ids == nil {
		return []int{}, nil
	}
	return stringslice.StringSliceToIntSlice(ids)
}

func cleanPaths(paths []string) []string {
	ret := []string{}
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p != "" {
			ret = append(ret, p)
		}
	}
	return ret
}

func (r *mutationResolver) UserTierCreate(ctx context.Context, input UserTierCreateInput) (*models.UserTier, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("tier name is required")
	}
	blocked, err := tagIDs(input.BlockedTagIds)
	if err != nil {
		return nil, err
	}
	required, err := tagIDs(input.RequiredTagIds)
	if err != nil {
		return nil, err
	}
	t := &models.UserTier{
		Name:           name,
		AllowedPaths:   cleanPaths(input.AllowedPaths),
		BlockedTagIDs:  blocked,
		RequiredTagIDs: required,
		CanDownload:    input.CanDownload == nil || *input.CanDownload,
		CanViewReels:   input.CanViewReels == nil || *input.CanViewReels,
	}
	if input.Description != nil {
		t.Description = *input.Description
	}
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		return r.repository.User.CreateTier(ctx, t)
	}); err != nil {
		return nil, err
	}
	invalidateUsers()
	return t, nil
}

func (r *mutationResolver) UserTierUpdate(ctx context.Context, input UserTierUpdateInput) (*models.UserTier, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	id, err := strconv.Atoi(input.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid id %q", input.ID)
	}
	translator := changesetTranslator{inputMap: getUpdateInputMap(ctx)}

	var t *models.UserTier
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		t, err = r.repository.User.FindTier(ctx, id)
		if err != nil {
			return err
		}
		if t == nil {
			return fmt.Errorf("tier %d not found", id)
		}
		if input.Name != nil {
			name := strings.TrimSpace(*input.Name)
			if name == "" {
				return errors.New("tier name is required")
			}
			t.Name = name
		}
		if input.Description != nil {
			t.Description = *input.Description
		}
		if translator.hasField("allowed_paths") {
			t.AllowedPaths = cleanPaths(input.AllowedPaths)
		}
		if translator.hasField("blocked_tag_ids") {
			if t.BlockedTagIDs, err = tagIDs(input.BlockedTagIds); err != nil {
				return err
			}
		}
		if translator.hasField("required_tag_ids") {
			if t.RequiredTagIDs, err = tagIDs(input.RequiredTagIds); err != nil {
				return err
			}
		}
		if input.CanDownload != nil {
			t.CanDownload = *input.CanDownload
		}
		if input.CanViewReels != nil {
			t.CanViewReels = *input.CanViewReels
		}
		return r.repository.User.UpdateTier(ctx, t)
	}); err != nil {
		return nil, err
	}
	invalidateUsers()
	return t, nil
}

func (r *mutationResolver) UserTierDestroy(ctx context.Context, id string) (bool, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return false, err
	}
	tid, err := strconv.Atoi(id)
	if err != nil {
		return false, fmt.Errorf("invalid id %q", id)
	}
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		return r.repository.User.DestroyTier(ctx, tid)
	}); err != nil {
		return false, err
	}
	invalidateUsers()
	return true, nil
}

func (r *mutationResolver) ChangeMyPassword(ctx context.Context, currentPassword string, newPassword string) (bool, error) {
	p := access.FromContext(ctx)
	if p == nil {
		return false, errors.New("not logged in")
	}
	if p.IsOwner() {
		return false, errors.New("the owner password is changed in Settings > Security")
	}
	hash, err := manager.HashPassword(newPassword)
	if err != nil {
		return false, err
	}
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		u, err := r.repository.User.Find(ctx, p.UserID)
		if err != nil {
			return err
		}
		if u == nil {
			return errors.New("user not found")
		}
		if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(currentPassword)) != nil {
			return errors.New("current password is incorrect")
		}
		u.PasswordHash = hash
		return r.repository.User.Update(ctx, u)
	}); err != nil {
		return false, err
	}
	return true, nil
}
