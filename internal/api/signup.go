package api

// Stasharr: making an account with an invite link. The link is the login
// page with ?invite=CODE; the page asks the server whether the code is
// good, then sends the new username and password here.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sqlite"
)

type inviteInfo struct {
	OK     bool   `json:"ok"`
	Reason string `json:"reason,omitempty"`
	Tier   string `json:"tier,omitempty"`
	Site   string `json:"site,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

// findInvite looks up a usable invite and its tier's name.
func findInvite(ctx context.Context, repo models.Repository, code string) (*sqlite.Invite, string, string) {
	code = strings.TrimSpace(code)
	if code == "" || !config.GetInstance().HasCredentials() {
		return nil, "", "This invite link is not valid."
	}
	var inv *sqlite.Invite
	tierName := ""
	if err := repo.WithReadTxn(ctx, func(ctx context.Context) error {
		var err error
		inv, err = sqlite.FindInviteByCode(ctx, code)
		if err != nil || inv == nil || inv.TierID == nil {
			return err
		}
		t, err := repo.User.FindTier(ctx, *inv.TierID)
		if err == nil && t != nil {
			tierName = t.Name
		}
		return err
	}); err != nil {
		logger.Errorf("[invites] %v", err)
		return nil, "", "Something went wrong. Try again later."
	}
	if inv == nil {
		return nil, "", "This invite link is not valid."
	}
	if !inv.Usable() {
		return nil, "", "This invite link has been used up or has expired."
	}
	return inv, tierName, ""
}

func handleInviteInfo(repo models.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		addr := clientAddress(r)
		if manager.Guard.Blocked(addr, "invite") > 0 {
			writeJSON(w, http.StatusTooManyRequests, inviteInfo{Reason: "Too many tries. Try again later."})
			return
		}
		inv, tier, reason := findInvite(r.Context(), repo, r.URL.Query().Get("code"))
		if inv == nil {
			manager.Guard.Failed(addr, "invite")
			writeJSON(w, http.StatusOK, inviteInfo{Reason: reason})
			return
		}
		writeJSON(w, http.StatusOK, inviteInfo{OK: true, Tier: tier, Site: config.GetInstance().GetSiteName()})
	}
}

func handleSignup(repo models.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		addr := clientAddress(r)
		if wait := manager.Guard.Blocked(addr, "invite"); wait > 0 {
			http.Error(w, "Too many tries. Try again later.", http.StatusTooManyRequests)
			return
		}
		inv, _, reason := findInvite(r.Context(), repo, r.FormValue("invite"))
		if inv == nil {
			manager.Guard.Failed(addr, "invite")
			http.Error(w, reason, http.StatusBadRequest)
			return
		}
		us := manager.GetInstance().UserService
		username := strings.TrimSpace(r.FormValue("username"))
		if err := us.ValidateUsername(username); err != nil {
			http.Error(w, upperFirst(err.Error()), http.StatusBadRequest)
			return
		}
		if us.IsOwner(username) {
			http.Error(w, "That username is taken.", http.StatusBadRequest)
			return
		}
		hash, err := manager.HashPassword(r.FormValue("password"))
		if err != nil {
			http.Error(w, upperFirst(err.Error()), http.StatusBadRequest)
			return
		}
		role := access.Role(inv.Role)
		if !role.IsValid() || role == access.RoleAdmin {
			role = access.RoleViewer
		}
		u := &models.User{Username: username, PasswordHash: hash, Role: role, TierID: inv.TierID}
		taken := false
		used := true
		if err := repo.WithTxn(r.Context(), func(ctx context.Context) error {
			existing, err := repo.User.FindByUsername(ctx, username)
			if err != nil {
				return err
			}
			if existing != nil {
				taken = true
				return nil
			}
			used, err = sqlite.UseInvite(ctx, inv.ID)
			if err != nil || !used {
				return err
			}
			return repo.User.Create(ctx, u)
		}); err != nil {
			logger.Errorf("[invites] making an account: %v", err)
			http.Error(w, "Something went wrong. Try again later.", http.StatusInternalServerError)
			return
		}
		if taken {
			http.Error(w, "That username is taken.", http.StatusBadRequest)
			return
		}
		if !used {
			http.Error(w, "This invite link has been used up.", http.StatusBadRequest)
			return
		}
		us.Invalidate()
		logger.Infof("[invites] %q made an account with an invite", username)
		manager.Notify(manager.EventAccounts, "New account", fmt.Sprintf("%s made an account with an invite link.", username))

		// sign in straight away
		if err := manager.GetInstance().SessionStore.Login(w, r); err != nil {
			logger.Warnf("[invites] signing in %q: %v", username, err)
		}
		w.WriteHeader(http.StatusOK)
	}
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:] + "."
}
