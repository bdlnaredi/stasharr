package api

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/static"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sqlite"
	"github.com/stashapp/stash/pkg/utils"
)

type TagFinder interface {
	models.TagGetter
	GetImage(ctx context.Context, tagID int) ([]byte, error)
}

type tagRoutes struct {
	routes
	tagFinder   TagFinder
	sceneFinder models.SceneReader
}

func (rs tagRoutes) Routes() chi.Router {
	r := chi.NewRouter()

	r.Route("/{tagId}", func(r chi.Router) {
		r.Use(rs.TagCtx)
		r.Get("/image", rs.Image)
		r.Get("/clip.mp4", rs.Clip)
		r.Get("/clip.jpg", rs.ClipStill)
	})

	return r
}

func (rs tagRoutes) Image(w http.ResponseWriter, r *http.Request) {
	tag := r.Context().Value(tagKey).(*models.Tag)
	defaultParam := r.URL.Query().Get("default")

	var image []byte
	if defaultParam != "true" {
		readTxnErr := rs.withReadTxn(r, func(ctx context.Context) error {
			var err error
			image, err = rs.tagFinder.GetImage(ctx, tag.ID)
			return err
		})
		if errors.Is(readTxnErr, context.Canceled) {
			return
		}
		if readTxnErr != nil {
			logger.Warnf("read transaction error on fetch tag image: %v", readTxnErr)
		}
	}

	// fallback to default image
	if len(image) == 0 {
		image = static.ReadAll(static.DefaultTagImage)
	}

	utils.ServeImage(w, r, image)
}

// tagClipFile returns the tag's clip file when the viewer may see the scene
// it was cut from.
func (rs tagRoutes) tagClipFile(r *http.Request, still bool) string {
	tag := r.Context().Value(tagKey).(*models.Tag)
	var path string
	_ = rs.withReadTxn(r, func(ctx context.Context) error {
		c, err := sqlite.FindTagClip(ctx, tag.ID)
		if err != nil || c == nil {
			return err
		}
		// scene lookups follow the viewer's tier, so a hidden scene is nil
		s, err := rs.sceneFinder.Find(ctx, c.SceneID)
		if err != nil || s == nil {
			return err
		}
		video, picture := manager.TagClipPaths(tag.ID)
		path = video
		if still {
			path = picture
		}
		return nil
	})
	if path == "" {
		return ""
	}
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return path
}

// Clip serves the tag's short looping clip.
func (rs tagRoutes) Clip(w http.ResponseWriter, r *http.Request) {
	p := rs.tagClipFile(r, false)
	if p == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeFile(w, r, p)
}

// ClipStill serves a still picture from the tag's clip.
func (rs tagRoutes) ClipStill(w http.ResponseWriter, r *http.Request) {
	p := rs.tagClipFile(r, true)
	if p == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeFile(w, r, p)
}

func (rs tagRoutes) TagCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tagID, err := strconv.Atoi(chi.URLParam(r, "tagId"))
		if err != nil {
			http.Error(w, http.StatusText(404), 404)
			return
		}

		var tag *models.Tag
		_ = rs.withReadTxn(r, func(ctx context.Context) error {
			var err error
			tag, err = rs.tagFinder.Find(ctx, tagID)
			return err
		})
		if tag == nil {
			http.Error(w, http.StatusText(404), 404)
			return
		}

		ctx := context.WithValue(r.Context(), tagKey, tag)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
