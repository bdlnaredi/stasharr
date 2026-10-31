package api

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sqlite"
)

// siteAppliesTo reports whether the site replaces the normal interface for
// the user making the request.
func siteAppliesTo(ctx context.Context, cfg *config.Config) bool {
	if !cfg.GetSiteEnabled() {
		return false
	}
	p := access.FromContext(ctx)
	if p == nil {
		// no login, or the owner account
		return false
	}
	switch p.Role {
	case access.RoleViewer:
		return true
	case access.RoleEditor:
		return cfg.GetSiteAudience() == "non_admins"
	default:
		return false
	}
}

func (r *Resolver) siteConfig(ctx context.Context) (*SiteConfig, error) {
	cfg := config.GetInstance()
	ret := &SiteConfig{
		Enabled:      cfg.GetSiteEnabled(),
		Audience:     cfg.GetSiteAudience(),
		Style:        cfg.GetSiteStyle(),
		Name:         cfg.GetSiteName(),
		Tagline:      cfg.GetSiteTagline(),
		Featured:     cfg.GetSiteFeatured(),
		ModelsGender: cfg.GetSiteModelsGender(),
		AppliesToMe:  siteAppliesTo(ctx, cfg),
	}
	if id := cfg.GetSiteFeaturedTag(); id > 0 {
		err := r.withReadTxn(ctx, func(ctx context.Context) error {
			t, err := r.repository.Tag.Find(ctx, id)
			ret.FeaturedTag = t
			return err
		})
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func (r *queryResolver) SiteConfig(ctx context.Context) (*SiteConfig, error) {
	return r.siteConfig(ctx)
}

func (r *mutationResolver) ConfigureSite(ctx context.Context, input SiteConfigInput) (*SiteConfig, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	cfg := config.GetInstance()
	if input.Enabled != nil {
		cfg.SetBool(config.SiteEnabled, *input.Enabled)
	}
	if input.Audience != nil {
		if *input.Audience != "viewers" && *input.Audience != "non_admins" {
			return nil, errors.New("audience must be viewers or non_admins")
		}
		cfg.SetString(config.SiteAudience, *input.Audience)
	}
	if input.Style != nil {
		if !slices.Contains(config.SiteStyles, *input.Style) {
			return nil, errors.New("unknown style")
		}
		cfg.SetString(config.SiteStyle, *input.Style)
	}
	if input.Name != nil {
		cfg.SetString(config.SiteName, strings.TrimSpace(*input.Name))
	}
	if input.Tagline != nil {
		cfg.SetString(config.SiteTagline, strings.TrimSpace(*input.Tagline))
	}
	if input.Featured != nil {
		switch *input.Featured {
		case "top_rated", "popular", "recent", "random", "tag":
		default:
			return nil, errors.New("unknown featured setting")
		}
		cfg.SetString(config.SiteFeatured, *input.Featured)
	}
	if input.FeaturedTagID != nil {
		id := 0
		if *input.FeaturedTagID != "" {
			n, err := strconv.Atoi(*input.FeaturedTagID)
			if err != nil {
				return nil, errors.New("invalid tag id")
			}
			id = n
		}
		cfg.SetInt(config.SiteFeaturedTag, id)
	}
	if input.ModelsGender != nil {
		g := "FEMALE"
		if *input.ModelsGender == "ALL" {
			g = "ALL"
		}
		cfg.SetString(config.SiteModelsGender, g)
	}
	if err := cfg.Write(); err != nil {
		return nil, err
	}
	return r.siteConfig(ctx)
}

func (r *queryResolver) SiteScenes(ctx context.Context, kind SiteSceneKind, page *int, perPage *int) (*SiteSceneResult, error) {
	n := 24
	if perPage != nil && *perPage > 0 && *perPage <= 120 {
		n = *perPage
	}
	p := 1
	if page != nil && *page > 1 {
		p = *page
	}

	cfg := config.GetInstance()
	sort := sqlite.SitePopular
	tagID := 0
	switch kind {
	case SiteSceneKindTrending:
		sort = sqlite.SiteTrending
	case SiteSceneKindTopRated:
		sort = sqlite.SiteTopRated
	case SiteSceneKindFeatured:
		switch cfg.GetSiteFeatured() {
		case "popular":
			sort = sqlite.SitePopular
		case "recent":
			sort = sqlite.SiteRecent
		case "random":
			sort = sqlite.SiteRandom
		case "tag":
			sort = sqlite.SiteRecent
			tagID = cfg.GetSiteFeaturedTag()
		default:
			sort = sqlite.SiteTopRated
		}
	}

	ret := &SiteSceneResult{Scenes: []*models.Scene{}}
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ids, total, err := sqlite.SiteSceneIDs(ctx, sort, tagID, n, (p-1)*n)
		if err != nil {
			return err
		}
		// nothing rated yet: show the newest instead of an empty banner
		if total == 0 && kind == SiteSceneKindFeatured {
			ids, total, err = sqlite.SiteSceneIDs(ctx, sqlite.SiteRecent, 0, n, (p-1)*n)
			if err != nil {
				return err
			}
		}
		ret.Count = total
		ret.Scenes, err = r.scenesInOrder(ctx, ids)
		return err
	})
	return ret, err
}
