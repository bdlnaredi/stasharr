package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

// Frame serves a picture of the scene at a moment, for search results. It
// is made once with ffmpeg and kept in the generated folder under frames/.
func (rs sceneRoutes) Frame(w http.ResponseWriter, r *http.Request) {
	scene := r.Context().Value(sceneKey).(*models.Scene)
	sec, err := strconv.ParseFloat(chi.URLParam(r, "sec"), 64)
	if err != nil || sec < 0 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	vf := scene.Files.Primary()
	if vf == nil || vf.IsAudioOnly() {
		http.NotFound(w, r)
		return
	}
	if sec > vf.Duration {
		sec = vf.Duration
	}
	sec = float64(int(sec))
	dir := filepath.Join(config.GetInstance().GetGeneratedPath(), "frames")
	path := filepath.Join(dir, fmt.Sprintf("%d_%d.jpg", scene.ID, int(sec)))
	if _, err := os.Stat(path); err != nil {
		if err := fsutil.EnsureDir(dir); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		if err := manager.SaveFrame(ctx, vf.Path, sec, path); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=604800")
	utils.ServeImage(w, r, b)
}
