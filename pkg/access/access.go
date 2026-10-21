// Package access defines the multi-user model used by this fork: roles,
// tiers (library access policies) and the per-request principal that is
// carried in the request context.
//
// A request without a principal (background jobs, the setup wizard, a
// system with no credentials configured) behaves exactly like upstream
// Stash: full access, global history and ratings.
package access

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Role controls which operations a user may perform.
type Role string

const (
	// RoleAdmin can do everything, including settings and user management.
	RoleAdmin Role = "ADMIN"
	// RoleEditor can edit metadata, scrape/identify and run library tasks,
	// but cannot change settings, manage users or delete files.
	RoleEditor Role = "EDITOR"
	// RoleViewer can browse and play content allowed by their tier, and keep
	// their own ratings, play history and O history.
	RoleViewer Role = "VIEWER"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleEditor, RoleViewer:
		return true
	}
	return false
}

// Rank orders roles from least to most privileged.
func (r Role) Rank() int {
	switch r {
	case RoleAdmin:
		return 3
	case RoleEditor:
		return 2
	case RoleViewer:
		return 1
	}
	return 0
}

// AtLeast reports whether r has at least the privileges of other.
func (r Role) AtLeast(other Role) bool {
	return r.Rank() >= other.Rank()
}

// ErrForbidden is returned when the current user may not perform an operation.
var ErrForbidden = errors.New("forbidden: your account does not have permission for this action")

// ErrHidden is returned by loaders for objects the current user's tier
// does not allow them to see.
var ErrHidden = errors.New("not found")

// Policy is the library access policy of a tier.
type Policy struct {
	// AllowedPaths restricts content to files under these library folders.
	// Empty means all paths.
	AllowedPaths []string
	// BlockedTagIDs hides any content tagged with one of these tags, and
	// hides the tags themselves.
	BlockedTagIDs []int
	// RequiredTagIDs, when non-empty, only shows content that has at least
	// one of these tags.
	RequiredTagIDs []int
	// CanDownload allows downloading original files.
	CanDownload bool
	// CanViewReels allows access to generated reels.
	CanViewReels bool
}

// Empty reports whether the policy places no restriction on content.
func (p Policy) Empty() bool {
	return len(p.AllowedPaths) == 0 && len(p.BlockedTagIDs) == 0 && len(p.RequiredTagIDs) == 0
}

// Principal is the authenticated user of a request.
type Principal struct {
	// UserID is the database id of the user. 0 means the owner account
	// configured in Settings > Security (the original single Stash user).
	UserID   int
	Username string
	Role     Role
	TierID   *int
	TierName string
	Policy   Policy
}

// IsOwner reports whether this is the built-in owner account.
func (p *Principal) IsOwner() bool {
	return p != nil && p.UserID == 0
}

// RestrictsContent reports whether content queries must be filtered for p.
// Admins are never restricted.
func (p *Principal) RestrictsContent() bool {
	if p == nil || p.Role == RoleAdmin {
		return false
	}
	return !p.Policy.Empty()
}

// CanDownload reports whether p may download original files.
func (p *Principal) CanDownload() bool {
	if p == nil || p.Role == RoleAdmin || p.TierID == nil {
		return true
	}
	return p.Policy.CanDownload
}

// CanViewReels reports whether p may view generated reels.
func (p *Principal) CanViewReels() bool {
	if p == nil || p.Role == RoleAdmin || p.TierID == nil {
		return true
	}
	return p.Policy.CanViewReels
}

// OwnerPrincipal returns the principal for the owner account.
func OwnerPrincipal(username string) *Principal {
	return &Principal{
		UserID:   0,
		Username: username,
		Role:     RoleAdmin,
		Policy:   Policy{CanDownload: true, CanViewReels: true},
	}
}

type contextKey struct{}

// WithPrincipal returns a context carrying p.
func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, p)
}

// FromContext returns the principal of the request, or nil.
func FromContext(ctx context.Context) *Principal {
	if ctx == nil {
		return nil
	}
	p, _ := ctx.Value(contextKey{}).(*Principal)
	return p
}

// HistoryUserID returns the user id whose personal history/ratings apply, or
// nil when the global (owner) data applies.
func HistoryUserID(ctx context.Context) *int {
	p := FromContext(ctx)
	if p == nil || p.UserID == 0 {
		return nil
	}
	id := p.UserID
	return &id
}

// ContentPolicy returns the policy that must be applied to content queries,
// or nil when no restriction applies.
func ContentPolicy(ctx context.Context) *Policy {
	p := FromContext(ctx)
	if !p.RestrictsContent() {
		return nil
	}
	return &p.Policy
}

// RequireRole returns ErrForbidden when the request's principal has less
// than the given role. Requests without a principal are allowed.
func RequireRole(ctx context.Context, role Role) error {
	p := FromContext(ctx)
	if p == nil {
		return nil
	}
	if !p.Role.AtLeast(role) {
		return ErrForbidden
	}
	return nil
}

// PathAllowed reports whether a file path is inside one of the allowed paths.
func (p Policy) PathAllowed(path string) bool {
	if len(p.AllowedPaths) == 0 {
		return true
	}
	for _, allowed := range p.AllowedPaths {
		a := strings.TrimRight(allowed, `/\`)
		if path == a || strings.HasPrefix(path, a+"/") || strings.HasPrefix(path, a+`\`) {
			return true
		}
	}
	return false
}

// MarshalGQL implements graphql.Marshaler.
func (r Role) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(string(r)))
}

// UnmarshalGQL implements graphql.Unmarshaler.
func (r *Role) UnmarshalGQL(v interface{}) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("role must be a string")
	}
	*r = Role(s)
	if !r.IsValid() {
		return fmt.Errorf("%s is not a valid UserRole", s)
	}
	return nil
}
