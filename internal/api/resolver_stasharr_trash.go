package api

import (
	"context"
	"errors"
	"time"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/trash"
)

func trashConfig() *TrashConfig {
	cfg := manager.GetInstance().Config
	return &TrashConfig{
		Enabled:       cfg.GetTrashEnabled(),
		RetentionDays: cfg.GetTrashRetentionDays(),
		TrashPath:     cfg.GetDeleteTrashPath(),
	}
}

func (r *queryResolver) TrashBin(ctx context.Context) (*TrashBin, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	entries, err := trash.List(manager.TrashDirs())
	if err != nil {
		return nil, err
	}
	cfg := trashConfig()
	ret := &TrashBin{Config: cfg, Items: []*TrashItem{}}
	for _, e := range entries {
		item := &TrashItem{
			ID:           e.ID,
			OriginalPath: e.OriginalPath,
			TrashPath:    e.TrashPath,
			Size:         e.Size,
			IsDir:        e.IsDir,
			DeletedAt:    e.DeletedAt,
		}
		if cfg.RetentionDays > 0 {
			at := e.DeletedAt.Add(time.Duration(cfg.RetentionDays) * 24 * time.Hour)
			item.PurgeAt = &at
		}
		ret.TotalSize += e.Size
		ret.Items = append(ret.Items, item)
	}
	return ret, nil
}

func (r *mutationResolver) ConfigureTrash(ctx context.Context, input TrashConfigInput) (*TrashConfig, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	cfg := manager.GetInstance().Config
	if input.Enabled != nil {
		cfg.SetBool(config.TrashEnabled, *input.Enabled)
	}
	if input.RetentionDays != nil {
		if *input.RetentionDays < 0 || *input.RetentionDays > 3650 {
			return nil, errors.New("days must be between 0 and 3650")
		}
		cfg.SetInt(config.TrashRetentionDays, *input.RetentionDays)
	}
	if err := cfg.Write(); err != nil {
		return nil, err
	}
	go manager.PurgeTrash()
	return trashConfig(), nil
}

func (r *mutationResolver) TrashRestore(ctx context.Context, ids []string) ([]string, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	ret := []string{}
	var errs []error
	for _, id := range ids {
		p, err := manager.RestoreFromTrash(context.WithoutCancel(access.WithPrincipal(ctx, nil)), id)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		ret = append(ret, p)
	}
	if len(ret) == 0 && len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return ret, nil
}

func (r *mutationResolver) TrashDelete(ctx context.Context, ids []string) (int, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		// make sure files not yet in the index are emptied too
		if _, err := trash.List(manager.TrashDirs()); err != nil {
			return 0, err
		}
	}
	return trash.Delete(ids)
}
