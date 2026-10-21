package api

import (
	"context"
	"errors"
	"sort"
	"strconv"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
)

// --- duplicate cleanup

func duplicateOptions(input DuplicateCleanupInput) (manager.DuplicateCleanupOptions, error) {
	opts := manager.DuplicateCleanupOptions{Distance: 0, DurationDiff: 1}
	if input.Distance != nil {
		if *input.Distance < 0 || *input.Distance > 32 {
			return opts, errors.New("distance must be between 0 and 32")
		}
		opts.Distance = *input.Distance
	}
	if input.DurationDiff != nil {
		opts.DurationDiff = *input.DurationDiff
	}
	ids, err := stringslice.StringSliceToIntSlice(input.KeepSceneIds)
	if err != nil {
		return opts, err
	}
	opts.KeepSceneIDs = ids
	return opts, nil
}

func (r *queryResolver) DuplicateCleanupPreview(ctx context.Context, input DuplicateCleanupInput) ([]*DuplicateCleanupGroup, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	opts, err := duplicateOptions(input)
	if err != nil {
		return nil, err
	}
	plans, err := manager.PlanDuplicateCleanup(access.WithPrincipal(ctx, nil), opts)
	if err != nil {
		return nil, err
	}
	ret := make([]*DuplicateCleanupGroup, 0, len(plans))
	for _, p := range plans {
		ret = append(ret, &DuplicateCleanupGroup{Keep: p.Keep, Remove: p.Remove})
	}
	return ret, nil
}

func (r *mutationResolver) DuplicateCleanup(ctx context.Context, input DuplicateCleanupInput) (string, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return "", err
	}
	opts, err := duplicateOptions(input)
	if err != nil {
		return "", err
	}
	j := &manager.DuplicateCleanupJob{Options: opts}
	jobID := manager.GetInstance().JobManager.Add(context.WithoutCancel(access.WithPrincipal(ctx, nil)), "Cleaning up duplicates...", j)
	return strconv.Itoa(jobID), nil
}

// --- renamer

func renamerConfig() *RenamerConfig {
	cfg := manager.GetInstance().Config
	ret := &RenamerConfig{
		Enabled:         cfg.GetRenamerEnabled(),
		Template:        cfg.GetRenamerTemplate(),
		DefaultTemplate: config.DefaultRenamerTemplate,
		Tokens:          []*RenamerToken{},
	}
	for _, t := range manager.RenamerTokens {
		ret.Tokens = append(ret.Tokens, &RenamerToken{Token: t[0], Description: t[1]})
	}
	return ret
}

func (r *queryResolver) RenamerConfig(ctx context.Context) (*RenamerConfig, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	return renamerConfig(), nil
}

func (r *mutationResolver) ConfigureRenamer(ctx context.Context, input RenamerConfigInput) (*RenamerConfig, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	cfg := manager.GetInstance().Config
	if input.Template != nil {
		if err := manager.ValidateRenamerTemplate(*input.Template); err != nil {
			return nil, err
		}
		cfg.SetString(config.RenamerTemplate, *input.Template)
	}
	if input.Enabled != nil {
		cfg.SetBool(config.RenamerEnabled, *input.Enabled)
	}
	if err := cfg.Write(); err != nil {
		return nil, err
	}
	return renamerConfig(), nil
}

const renamePreviewLimit = 200

func (r *queryResolver) RenamePreview(ctx context.Context, input RenameScenesInput) (*RenamePreviewResult, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	ids, err := stringslice.StringSliceToIntSlice(input.SceneIds)
	if err != nil {
		return nil, err
	}
	tmpl := ""
	if input.Template != nil {
		tmpl = *input.Template
	}
	plans, err := manager.PlanRenames(access.WithPrincipal(ctx, nil), ids, tmpl, 0)
	if err != nil {
		return nil, err
	}

	// files that would move first, then skipped ones, then the rest
	rank := map[string]int{manager.RenameMove: 0, manager.RenameSkip: 1, manager.RenameSame: 2}
	sort.SliceStable(plans, func(i, j int) bool { return rank[plans[i].Status] < rank[plans[j].Status] })

	ret := &RenamePreviewResult{Checked: len(plans), Items: []*RenamePreviewItem{}}
	for _, p := range plans {
		if p.Status == manager.RenameMove {
			ret.ToRename++
		}
		if len(ret.Items) >= renamePreviewLimit {
			continue
		}
		ret.Items = append(ret.Items, &RenamePreviewItem{
			SceneID: strconv.Itoa(p.SceneID),
			Title:   p.Title,
			OldPath: p.OldPath,
			NewPath: p.NewPath,
			Status:  p.Status,
			Reason:  p.Reason,
		})
	}
	return ret, nil
}

func (r *mutationResolver) RenameScenes(ctx context.Context, input RenameScenesInput) (string, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return "", err
	}
	ids, err := stringslice.StringSliceToIntSlice(input.SceneIds)
	if err != nil {
		return "", err
	}
	j := &manager.RenameJob{SceneIDs: ids}
	if input.Template != nil {
		if err := manager.ValidateRenamerTemplate(*input.Template); err != nil {
			return "", err
		}
		j.Template = *input.Template
	}
	jobID := manager.GetInstance().JobManager.Add(context.WithoutCancel(access.WithPrincipal(ctx, nil)), "Renaming files...", j)
	return strconv.Itoa(jobID), nil
}
