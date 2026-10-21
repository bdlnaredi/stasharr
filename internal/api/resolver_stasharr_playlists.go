package api

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stashapp/stash/pkg/sqlite"
)

func (r *Resolver) Playlist() PlaylistResolver { return &playlistResolver{r} }

type playlistResolver struct{ *Resolver }

var errPlaylistNotFound = errors.New("playlist not found")

// scenesInOrder loads scenes by id in the given order, skipping any the
// caller cannot see.
func (r *Resolver) scenesInOrder(ctx context.Context, ids []int) ([]*models.Scene, error) {
	if len(ids) == 0 {
		return []*models.Scene{}, nil
	}
	found, err := r.repository.Scene.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[int]*models.Scene, len(found))
	for _, s := range found {
		byID[s.ID] = s
	}
	ret := make([]*models.Scene, 0, len(ids))
	for _, id := range ids {
		if s := byID[id]; s != nil {
			ret = append(ret, s)
		}
	}
	return ret, nil
}

func (r *playlistResolver) SceneCount(ctx context.Context, obj *models.Playlist) (n int, err error) {
	err = r.withReadTxn(ctx, func(ctx context.Context) error {
		n, err = r.repository.Playlist.SceneCount(ctx, obj.ID)
		return err
	})
	return n, err
}

func (r *playlistResolver) Cover(ctx context.Context, obj *models.Playlist) (*models.Scene, error) {
	var ret *models.Scene
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ids, err := r.repository.Playlist.SceneIDs(ctx, obj.ID)
		if err != nil || len(ids) == 0 {
			return err
		}
		scenes, err := r.scenesInOrder(ctx, ids[:1])
		if err != nil {
			return err
		}
		if len(scenes) > 0 {
			ret = scenes[0]
		}
		return nil
	})
	return ret, err
}

func (r *playlistResolver) Scenes(ctx context.Context, obj *models.Playlist) ([]*models.Scene, error) {
	var ret []*models.Scene
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ids, err := r.repository.Playlist.SceneIDs(ctx, obj.ID)
		if err != nil {
			return err
		}
		ret, err = r.scenesInOrder(ctx, ids)
		return err
	})
	if ret == nil {
		ret = []*models.Scene{}
	}
	return ret, err
}

func (r *sceneResolver) InWatchLater(ctx context.Context, obj *models.Scene) (bool, error) {
	uid := sqlite.PlaylistUserID(ctx)
	var in bool
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ids, err := r.repository.Playlist.PlaylistIDsWithScene(ctx, uid, obj.ID)
		if err != nil || len(ids) == 0 {
			return err
		}
		for _, id := range ids {
			p, err := r.repository.Playlist.Find(ctx, id)
			if err != nil {
				return err
			}
			if p != nil && p.WatchLater {
				in = true
			}
		}
		return nil
	})
	return in, err
}

func (r *sceneResolver) PlaylistIds(ctx context.Context, obj *models.Scene) ([]string, error) {
	uid := sqlite.PlaylistUserID(ctx)
	ret := []string{}
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ids, err := r.repository.Playlist.PlaylistIDsWithScene(ctx, uid, obj.ID)
		if err != nil {
			return err
		}
		for _, id := range ids {
			ret = append(ret, strconv.Itoa(id))
		}
		return nil
	})
	return ret, err
}

func (r *queryResolver) FindPlaylists(ctx context.Context) ([]*models.Playlist, error) {
	uid := sqlite.PlaylistUserID(ctx)
	var ret []*models.Playlist
	// creates the Watch later playlist the first time
	err := r.withTxn(ctx, func(ctx context.Context) error {
		if _, err := r.repository.Playlist.WatchLater(ctx, uid); err != nil {
			return err
		}
		var err error
		ret, err = r.repository.Playlist.FindByUser(ctx, uid)
		return err
	})
	if ret == nil {
		ret = []*models.Playlist{}
	}
	return ret, err
}

// ownPlaylist loads a playlist of the current user. Must run in a transaction.
func (r *Resolver) ownPlaylist(ctx context.Context, id string) (*models.Playlist, error) {
	pid, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id %q", id)
	}
	p, err := r.repository.Playlist.Find(ctx, pid)
	if err != nil {
		return nil, err
	}
	if p == nil || p.UserID != sqlite.PlaylistUserID(ctx) {
		return nil, errPlaylistNotFound
	}
	return p, nil
}

func (r *queryResolver) FindPlaylist(ctx context.Context, id string) (*models.Playlist, error) {
	var ret *models.Playlist
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		var err error
		ret, err = r.ownPlaylist(ctx, id)
		if errors.Is(err, errPlaylistNotFound) {
			ret = nil
			return nil
		}
		return err
	})
	return ret, err
}

func (r *queryResolver) ContinueWatching(ctx context.Context, limit *int) ([]*models.Scene, error) {
	n := 20
	if limit != nil && *limit > 0 && *limit <= 100 {
		n = *limit
	}
	uid := sqlite.PlaylistUserID(ctx)
	var ret []*models.Scene
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ids, err := r.repository.Playlist.ContinueWatching(ctx, uid, n)
		if err != nil {
			return err
		}
		ret, err = r.scenesInOrder(ctx, ids)
		return err
	})
	if ret == nil {
		ret = []*models.Scene{}
	}
	return ret, err
}

func playlistName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("the playlist needs a name")
	}
	if len(name) > 200 {
		return "", errors.New("the name is too long")
	}
	if strings.EqualFold(name, sqlite.WatchLaterName) {
		return "", errors.New("that name is used by the Watch later playlist")
	}
	return name, nil
}

func (r *mutationResolver) PlaylistCreate(ctx context.Context, input PlaylistCreateInput) (*models.Playlist, error) {
	name, err := playlistName(input.Name)
	if err != nil {
		return nil, err
	}
	sceneIDs, err := stringslice.StringSliceToIntSlice(input.SceneIds)
	if err != nil {
		return nil, err
	}
	p := &models.Playlist{UserID: sqlite.PlaylistUserID(ctx), Name: name}
	if input.Description != nil {
		p.Description = strings.TrimSpace(*input.Description)
	}
	err = r.withTxn(ctx, func(ctx context.Context) error {
		if err := r.repository.Playlist.Create(ctx, p); err != nil {
			return err
		}
		visible, err := r.visibleSceneIDs(ctx, sceneIDs)
		if err != nil {
			return err
		}
		return r.repository.Playlist.AddScenes(ctx, p.ID, visible)
	})
	return p, err
}

// visibleSceneIDs drops scenes the current user's tier hides.
func (r *Resolver) visibleSceneIDs(ctx context.Context, ids []int) ([]int, error) {
	scenes, err := r.scenesInOrder(ctx, ids)
	if err != nil {
		return nil, err
	}
	ret := make([]int, len(scenes))
	for i, s := range scenes {
		ret[i] = s.ID
	}
	return ret, nil
}

func (r *mutationResolver) PlaylistUpdate(ctx context.Context, input PlaylistUpdateInput) (*models.Playlist, error) {
	var p *models.Playlist
	err := r.withTxn(ctx, func(ctx context.Context) error {
		var err error
		p, err = r.ownPlaylist(ctx, input.ID)
		if err != nil {
			return err
		}
		if input.Name != nil {
			if p.WatchLater {
				return errors.New("Watch later cannot be renamed")
			}
			name, err := playlistName(*input.Name)
			if err != nil {
				return err
			}
			p.Name = name
		}
		if input.Description != nil {
			p.Description = strings.TrimSpace(*input.Description)
		}
		return r.repository.Playlist.Update(ctx, p)
	})
	return p, err
}

func (r *mutationResolver) PlaylistDestroy(ctx context.Context, id string) (bool, error) {
	err := r.withTxn(ctx, func(ctx context.Context) error {
		p, err := r.ownPlaylist(ctx, id)
		if err != nil {
			return err
		}
		if p.WatchLater {
			return errors.New("Watch later cannot be deleted")
		}
		return r.repository.Playlist.Destroy(ctx, p.ID)
	})
	return err == nil, err
}

func (r *mutationResolver) playlistChange(ctx context.Context, id string, sceneIDs []string, fn func(ctx context.Context, p *models.Playlist, ids []int) error) (*models.Playlist, error) {
	ids, err := stringslice.StringSliceToIntSlice(sceneIDs)
	if err != nil {
		return nil, err
	}
	var p *models.Playlist
	err = r.withTxn(ctx, func(ctx context.Context) error {
		var err error
		p, err = r.ownPlaylist(ctx, id)
		if err != nil {
			return err
		}
		if err := fn(ctx, p, ids); err != nil {
			return err
		}
		p, err = r.repository.Playlist.Find(ctx, p.ID)
		return err
	})
	return p, err
}

func (r *mutationResolver) PlaylistAddScenes(ctx context.Context, id string, sceneIds []string) (*models.Playlist, error) {
	return r.playlistChange(ctx, id, sceneIds, func(ctx context.Context, p *models.Playlist, ids []int) error {
		visible, err := r.visibleSceneIDs(ctx, ids)
		if err != nil {
			return err
		}
		return r.repository.Playlist.AddScenes(ctx, p.ID, visible)
	})
}

func (r *mutationResolver) PlaylistRemoveScenes(ctx context.Context, id string, sceneIds []string) (*models.Playlist, error) {
	return r.playlistChange(ctx, id, sceneIds, func(ctx context.Context, p *models.Playlist, ids []int) error {
		return r.repository.Playlist.RemoveScenes(ctx, p.ID, ids)
	})
}

func (r *mutationResolver) PlaylistReorder(ctx context.Context, id string, sceneIds []string) (*models.Playlist, error) {
	return r.playlistChange(ctx, id, sceneIds, func(ctx context.Context, p *models.Playlist, ids []int) error {
		return r.repository.Playlist.SetOrder(ctx, p.ID, ids)
	})
}

func (r *mutationResolver) WatchLaterToggle(ctx context.Context, sceneID string) (bool, error) {
	sid, err := strconv.Atoi(sceneID)
	if err != nil {
		return false, fmt.Errorf("invalid id %q", sceneID)
	}
	uid := sqlite.PlaylistUserID(ctx)
	added := false
	err = r.withTxn(ctx, func(ctx context.Context) error {
		wl, err := r.repository.Playlist.WatchLater(ctx, uid)
		if err != nil {
			return err
		}
		in, err := r.repository.Playlist.PlaylistIDsWithScene(ctx, uid, sid)
		if err != nil {
			return err
		}
		for _, id := range in {
			if id == wl.ID {
				return r.repository.Playlist.RemoveScenes(ctx, wl.ID, []int{sid})
			}
		}
		visible, err := r.visibleSceneIDs(ctx, []int{sid})
		if err != nil {
			return err
		}
		if len(visible) == 0 {
			return errors.New("scene not found")
		}
		added = true
		return r.repository.Playlist.AddScenes(ctx, wl.ID, visible)
	})
	return added, err
}
