package api

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/models"
)

type reelRoutes struct {
	repository models.Repository
}

func (rs reelRoutes) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{reelId}/stream.mp4", rs.Stream)
	r.Get("/{reelId}/stream", rs.Stream)
	r.Get("/{reelId}/cover.jpg", rs.Cover)
	return r
}

func (rs reelRoutes) Stream(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !access.FromContext(ctx).CanViewReels() {
		http.Error(w, "reels are not included in your tier", http.StatusForbidden)
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "reelId"))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var reel *models.Reel
	if err := rs.repository.WithReadTxn(ctx, func(ctx context.Context) error {
		var err error
		reel, err = rs.repository.Reel.Find(ctx, id)
		return err
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if reel == nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeFile(w, r, reel.Path)
}

// reelCoverLocks keeps two requests from making the same picture at once.
var reelCoverLocks sync.Map

// Cover serves the reel's picture, making it first for reels made before
// pictures were.
func (rs reelRoutes) Cover(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !access.FromContext(ctx).CanViewReels() {
		http.Error(w, "reels are not included in your tier", http.StatusForbidden)
		return
	}
	id, err := strconv.Atoi(chi.URLParam(r, "reelId"))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	var reel *models.Reel
	if err := rs.repository.WithReadTxn(ctx, func(ctx context.Context) error {
		var err error
		reel, err = rs.repository.Reel.Find(ctx, id)
		return err
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if reel == nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	cover := manager.ReelCoverPath(reel.Path)
	if _, err := os.Stat(cover); err != nil {
		lock, _ := reelCoverLocks.LoadOrStore(id, &sync.Mutex{})
		mu := lock.(*sync.Mutex)
		mu.Lock()
		if _, err := os.Stat(cover); err != nil {
			err = manager.MakeReelCover(ctx, reel.Path, reel.Duration)
			if err != nil {
				mu.Unlock()
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		mu.Unlock()
	}
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeFile(w, r, cover)
}

func (s *Server) getReelRoutes() chi.Router {
	return reelRoutes{repository: s.manager.Repository}.Routes()
}
