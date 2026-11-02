package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sqlite"
)

// The login page shows pictures or short previews from the library behind
// the form. Visitors are not signed in, so the pictures are addressed by
// their place in a small list that is picked again every so often, never by
// scene id.

const (
	backdropCount   = 12
	backdropRefresh = 20 * time.Minute
)

type loginBackdrop struct {
	repository models.Repository

	mu     sync.Mutex
	ids    []int
	videos map[int]bool
	at     time.Time
	key    string
}

type backdropItem struct {
	Image string `json:"image"`
	Video string `json:"video,omitempty"`
}

type backdropInfo struct {
	Mode  string         `json:"mode"`
	Name  string         `json:"name"`
	Logo  bool           `json:"logo"`
	Items []backdropItem `json:"items"`
}

func previewPath(s *models.Scene) string {
	hash := s.GetHash(config.GetInstance().GetVideoFileNamingAlgorithm())
	return manager.GetInstance().Paths.Scene.GetVideoPreviewPath(hash)
}

// pick returns the current list, picking a new one when it is old or the
// settings changed.
func (b *loginBackdrop) pick(ctx context.Context) ([]int, map[int]bool) {
	cfg := config.GetInstance()
	key := cfg.GetLoginBackdrop() + "/" + strconv.Itoa(cfg.GetLoginBackdropTag())

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.key == key && time.Since(b.at) < backdropRefresh && b.ids != nil {
		return b.ids, b.videos
	}

	ids := []int{}
	videos := map[int]bool{}
	if cfg.GetLoginBackdrop() != "off" {
		err := b.repository.WithReadTxn(ctx, func(ctx context.Context) error {
			picked, err := sqlite.LoginBackdropSceneIDs(ctx, cfg.GetLoginBackdropTag(), backdropCount*2)
			if err != nil {
				return err
			}
			scenes, err := b.repository.Scene.FindMany(ctx, picked)
			if err != nil {
				return err
			}
			for _, s := range scenes {
				if s == nil || len(ids) >= backdropCount {
					continue
				}
				if err := s.LoadPrimaryFile(ctx, b.repository.File); err != nil {
					continue
				}
				if _, err := os.Stat(previewPath(s)); err == nil {
					videos[s.ID] = true
				}
				ids = append(ids, s.ID)
			}
			return nil
		})
		if err != nil {
			logger.Warnf("login page pictures: %v", err)
		}
	}
	b.ids, b.videos, b.at, b.key = ids, videos, time.Now(), key
	return ids, videos
}

func (b *loginBackdrop) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", b.Info)
	r.Get("/{n}.jpg", b.Picture)
	r.Get("/{n}.mp4", b.Preview)
	r.Get("/logo", siteLogoHandler)
	return r
}

// Info lists what to show: the backdrop setting, the site's name and the
// picture and preview addresses.
func (b *loginBackdrop) Info(w http.ResponseWriter, r *http.Request) {
	cfg := config.GetInstance()
	mode := cfg.GetLoginBackdrop()
	ids, videos := b.pick(r.Context())
	info := backdropInfo{Mode: mode, Name: "Stasharr", Items: []backdropItem{}}
	if cfg.GetSiteEnabled() && strings.TrimSpace(cfg.GetSiteName()) != "" {
		info.Name = strings.TrimSpace(cfg.GetSiteName())
	}
	if _, ok := siteLogoFile(cfg); ok && cfg.GetSiteEnabled() {
		info.Logo = true
	}
	stamp := strconv.FormatInt(b.at.Unix(), 36)
	for i, id := range ids {
		it := backdropItem{Image: "login/backdrop/" + strconv.Itoa(i) + ".jpg?v=" + stamp}
		if mode == "previews" && videos[id] {
			it.Video = "login/backdrop/" + strconv.Itoa(i) + ".mp4?v=" + stamp
		}
		info.Items = append(info.Items, it)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(info)
}

func (b *loginBackdrop) scene(r *http.Request) *models.Scene {
	n, err := strconv.Atoi(chi.URLParam(r, "n"))
	if err != nil || n < 0 {
		return nil
	}
	ids, _ := b.pick(r.Context())
	if n >= len(ids) {
		return nil
	}
	var s *models.Scene
	_ = b.repository.WithReadTxn(r.Context(), func(ctx context.Context) error {
		var err error
		s, err = b.repository.Scene.Find(ctx, ids[n])
		if err == nil && s != nil {
			err = s.LoadPrimaryFile(ctx, b.repository.File)
		}
		return err
	})
	return s
}

func (b *loginBackdrop) Picture(w http.ResponseWriter, r *http.Request) {
	if config.GetInstance().GetLoginBackdrop() == "off" {
		http.NotFound(w, r)
		return
	}
	s := b.scene(r)
	if s == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	ss := manager.SceneServer{TxnManager: b.repository.TxnManager, SceneCoverGetter: b.repository.Scene}
	ss.ServeScreenshot(s, w, r)
}

func (b *loginBackdrop) Preview(w http.ResponseWriter, r *http.Request) {
	if config.GetInstance().GetLoginBackdrop() != "previews" {
		http.NotFound(w, r)
		return
	}
	s := b.scene(r)
	if s == nil {
		http.NotFound(w, r)
		return
	}
	p := previewPath(s)
	if _, err := os.Stat(p); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, p)
}
