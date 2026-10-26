package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/session"
	"github.com/stashapp/stash/pkg/signedurl"
)

var errNoDownloads = errors.New("your account is not allowed to download")

func canDownload(ctx context.Context) bool {
	p := access.FromContext(ctx)
	return p == nil || p.CanDownload()
}

func (r *queryResolver) OfflineQualities(ctx context.Context) ([]*OfflineQualityOption, error) {
	ret := []*OfflineQualityOption{}
	for _, q := range manager.OfflineQualities {
		ret = append(ret, &OfflineQualityOption{ID: q.ID, Label: q.Label, Height: q.Height})
	}
	return ret, nil
}

// offlineScene loads a scene the caller can see, with its file.
func (r *Resolver) offlineScene(ctx context.Context, id string) (*models.Scene, error) {
	sid, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id %q", id)
	}
	var scene *models.Scene
	err = r.withReadTxn(ctx, func(ctx context.Context) error {
		s, err := r.repository.Scene.Find(ctx, sid)
		if err != nil || s == nil {
			return err
		}
		if err := s.LoadPrimaryFile(ctx, r.repository.File); err != nil {
			return err
		}
		scene = s
		return nil
	})
	if err != nil {
		return nil, err
	}
	if scene == nil {
		return nil, errors.New("scene not found")
	}
	return scene, nil
}

func offlineURL(ctx context.Context, sceneID int, quality string) (string, error) {
	baseURL, _ := ctx.Value(BaseURLCtxKey).(string)
	u, err := url.Parse(strings.TrimRight(baseURL, "/") + fmt.Sprintf("/scene/%d/offline.mp4", sceneID))
	if err != nil {
		return "", err
	}
	q := url.Values{}
	cfg := manager.GetInstance().Config
	if cfg.HasCredentials() {
		userID := session.GetCurrentUserID(ctx)
		if userID == nil {
			return "", errors.New("user ID not found")
		}
		q = signedParams(cfg, *userID, signedurl.DerivePrefix(u.Path))
	} else if key := cfg.GetAPIKey(); key != "" {
		q.Set("apikey", key)
	}
	q.Set("quality", quality)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (r *Resolver) offlineResult(ctx context.Context, scene *models.Scene, quality string, st *manager.OfflineState) (*OfflineDownload, error) {
	ret := &OfflineDownload{SceneID: strconv.Itoa(scene.ID), Quality: quality, Status: "NONE"}
	if st == nil {
		return ret, nil
	}
	ret.Status = st.Status
	ret.Progress = st.Progress
	ret.Size = int(st.Size)
	if st.Error != "" {
		e := st.Error
		ret.Error = &e
	}
	if st.Status == manager.OfflineReady {
		if f := scene.Files.Primary(); f != nil && quality == "original" {
			ret.Size = int(f.Size)
		}
		u, err := offlineURL(ctx, scene.ID, quality)
		if err != nil {
			return nil, err
		}
		ret.URL = &u
	}
	return ret, nil
}

func (r *queryResolver) OfflineDownload(ctx context.Context, sceneID string, quality string) (*OfflineDownload, error) {
	if !canDownload(ctx) {
		return nil, errNoDownloads
	}
	scene, err := r.offlineScene(ctx, sceneID)
	if err != nil {
		return nil, err
	}
	return r.offlineResult(ctx, scene, quality, manager.OfflineStatus(scene.ID, quality))
}

func (r *mutationResolver) PrepareOfflineDownload(ctx context.Context, sceneID string, quality string) (*OfflineDownload, error) {
	if !canDownload(ctx) {
		return nil, errNoDownloads
	}
	scene, err := r.offlineScene(ctx, sceneID)
	if err != nil {
		return nil, err
	}
	st, err := manager.PrepareOffline(scene, quality)
	if err != nil {
		return nil, err
	}
	return r.offlineResult(ctx, scene, quality, st)
}

// Offline serves a prepared download.
func (rs sceneRoutes) Offline(w http.ResponseWriter, r *http.Request) {
	if !canDownload(r.Context()) {
		http.Error(w, errNoDownloads.Error(), http.StatusForbidden)
		return
	}
	scene := r.Context().Value(sceneKey).(*models.Scene)
	f := scene.Files.Primary()
	if f == nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	quality := r.URL.Query().Get("quality")
	q, ok := manager.FindOfflineQuality(quality)
	if !ok {
		http.Error(w, "unknown quality", http.StatusBadRequest)
		return
	}

	name := scene.GetTitle()
	if name == "" {
		name = strings.TrimSuffix(f.Basename, filepath.Ext(f.Basename))
	}
	name = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`/\:*?"<>|`, r) || r < 32 {
			return '_'
		}
		return r
	}, name)

	path := f.Path
	ext := filepath.Ext(f.Basename)
	if q.Height > 0 {
		path = manager.OfflineFilePath(scene.ID, q.ID)
		st := manager.OfflineStatus(scene.ID, q.ID)
		if st == nil || st.Status != manager.OfflineReady {
			http.Error(w, "the download is not ready yet", http.StatusConflict)
			return
		}
		manager.TouchOffline(path)
		ext = ".mp4"
		w.Header().Set("Content-Type", "video/mp4")
	}
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(name+ext))
	http.ServeFile(w, r, path)
}
