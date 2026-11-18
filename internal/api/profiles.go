package api

// Stasharr: switching profile moves the browser session to the profile's
// account, after its PIN is checked.

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/logger"
)

func handleProfileSwitch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		reply := func(status int, msg string) {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": status == http.StatusOK, "error": msg})
		}

		// JSON only, so another site cannot send this from a plain form
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			reply(http.StatusUnsupportedMediaType, "Send JSON.")
			return
		}

		p := access.FromContext(r.Context())
		if p == nil {
			reply(http.StatusUnauthorized, "Sign in first.")
			return
		}
		var body struct {
			ID  string `json:"id"`
			Pin string `json:"pin"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil {
			reply(http.StatusBadRequest, "Bad request.")
			return
		}
		id, err := strconv.Atoi(body.ID)
		if err != nil {
			reply(http.StatusBadRequest, "Bad request.")
			return
		}

		username, err := manager.SwitchProfile(r.Context(), p, id, body.Pin)
		switch {
		case errors.Is(err, manager.ErrWrongPin):
			reply(http.StatusForbidden, "Wrong PIN.")
			return
		case errors.Is(err, manager.ErrPinLocked):
			reply(http.StatusTooManyRequests, err.Error())
			return
		case err != nil:
			reply(http.StatusBadRequest, err.Error())
			return
		}

		if err := manager.GetInstance().SessionStore.SwitchUser(w, r, username); err != nil {
			logger.Errorf("switching profile: %v", err)
			reply(http.StatusInternalServerError, "Could not switch profile.")
			return
		}
		reply(http.StatusOK, "")
	}
}
