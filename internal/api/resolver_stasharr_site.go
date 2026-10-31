package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
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
		AgeGate:      cfg.GetSiteAgeGate(),
		Accent:       cfg.GetSiteAccent(),
		AppLook:      cfg.GetSiteAppLook(),
		Promo:        sitePromo(cfg),
	}
	if p, ok := siteLogoFile(cfg); ok {
		if info, err := os.Stat(p); err == nil {
			baseURL, _ := ctx.Value(BaseURLCtxKey).(string)
			u := fmt.Sprintf("%s/site-logo?v=%d", strings.TrimRight(baseURL, "/"), info.ModTime().Unix())
			ret.LogoURL = &u
		}
	}
	rows, err := r.siteHomeRows(ctx, cfg)
	if err != nil {
		return nil, err
	}
	ret.HomeRows = rows
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
	if input.HomeRows != nil {
		var rows []storedHomeRow
		for _, in := range input.HomeRows {
			if in == nil || !slices.Contains(siteRowTypes, in.Type) {
				continue
			}
			row := storedHomeRow{Type: in.Type, Enabled: in.Enabled}
			if in.Title != nil {
				row.Title = strings.TrimSpace(*in.Title)
			}
			if in.TagID != nil && *in.TagID != "" {
				id, err := strconv.Atoi(*in.TagID)
				if err != nil {
					return nil, errors.New("invalid tag id")
				}
				row.TagID = id
			}
			if row.Type == "tag" && row.TagID == 0 {
				continue
			}
			rows = append(rows, row)
		}
		b, err := json.Marshal(rows)
		if err != nil {
			return nil, err
		}
		cfg.SetString(config.SiteHomeRows, string(b))
	}
	if input.Promo != nil {
		b, err := json.Marshal(storedPromo{
			Enabled: input.Promo.Enabled,
			Text:    strings.TrimSpace(input.Promo.Text),
			Link:    strings.TrimSpace(input.Promo.Link),
			Image:   strings.TrimSpace(input.Promo.Image),
		})
		if err != nil {
			return nil, err
		}
		cfg.SetString(config.SitePromo, string(b))
	}
	if input.AgeGate != nil {
		cfg.SetBool(config.SiteAgeGate, *input.AgeGate)
	}
	if input.AppLook != nil {
		cfg.SetBool(config.SiteAppLook, *input.AppLook)
	}
	if input.Accent != nil {
		a := strings.TrimSpace(*input.Accent)
		if a != "" && !isHexColour(a) {
			return nil, errors.New("the accent colour must look like #e84a7f")
		}
		cfg.SetString(config.SiteAccent, a)
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

func sitePage(page *int, perPage *int) (int, int) {
	n := 24
	if perPage != nil && *perPage > 0 && *perPage <= 120 {
		n = *perPage
	}
	p := 1
	if page != nil && *page > 1 {
		p = *page
	}
	return p, n
}

func (r *queryResolver) SiteScenes(ctx context.Context, kind SiteSceneKind, page *int, perPage *int, performerID *string) (*SiteSceneResult, error) {
	p, n := sitePage(page, perPage)
	performer := 0
	if performerID != nil && *performerID != "" {
		id, err := strconv.Atoi(*performerID)
		if err != nil {
			return nil, errors.New("invalid performer id")
		}
		performer = id
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
		filter := sqlite.SiteSceneFilter{TagID: tagID, PerformerID: performer}
		ids, total, err := sqlite.SiteSceneIDs(ctx, sort, filter, n, (p-1)*n)
		if err != nil {
			return err
		}
		// nothing rated yet: show the newest instead of an empty banner
		if total == 0 && kind == SiteSceneKindFeatured {
			ids, total, err = sqlite.SiteSceneIDs(ctx, sqlite.SiteRecent, sqlite.SiteSceneFilter{}, n, (p-1)*n)
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

// ---------------------------------------------------------------- home rows, promo, logo

// siteRowTypes are the kinds of rows the home page can show.
var siteRowTypes = []string{
	"continue", "latest", "models", "trending", "popular", "recommended",
	"favorites", "liked", "reels", "top_rated", "categories", "channels", "tag",
}

var defaultHomeRows = []string{
	"continue", "latest", "models", "trending", "favorites", "recommended",
	"reels", "top_rated", "categories", "channels",
}

type storedHomeRow struct {
	Type    string `json:"type"`
	Title   string `json:"title,omitempty"`
	TagID   int    `json:"tag_id,omitempty"`
	Enabled bool   `json:"enabled"`
}

type storedPromo struct {
	Enabled bool   `json:"enabled"`
	Text    string `json:"text"`
	Link    string `json:"link"`
	Image   string `json:"image"`
}

func sitePromo(cfg *config.Config) *SitePromo {
	var p storedPromo
	_ = json.Unmarshal([]byte(cfg.GetSitePromoJSON()), &p)
	return &SitePromo{Enabled: p.Enabled, Text: p.Text, Link: p.Link, Image: p.Image}
}

// siteHomeRows returns the saved rows, with any row type that is missing
// added at the end, switched off, so the admin can turn it on.
func (r *Resolver) siteHomeRows(ctx context.Context, cfg *config.Config) ([]*SiteHomeRow, error) {
	var stored []storedHomeRow
	if raw := cfg.GetSiteHomeRowsJSON(); raw != "" {
		_ = json.Unmarshal([]byte(raw), &stored)
	}
	if len(stored) == 0 {
		for _, t := range defaultHomeRows {
			stored = append(stored, storedHomeRow{Type: t, Enabled: true})
		}
	}
	have := map[string]bool{}
	for _, s := range stored {
		have[s.Type] = true
	}
	for _, t := range siteRowTypes {
		if t != "tag" && !have[t] {
			stored = append(stored, storedHomeRow{Type: t, Enabled: false})
		}
	}

	ret := make([]*SiteHomeRow, 0, len(stored))
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		for _, s := range stored {
			row := &SiteHomeRow{Type: s.Type, Title: s.Title, Enabled: s.Enabled}
			if s.Type == "tag" {
				t, err := r.repository.Tag.Find(ctx, s.TagID)
				if err != nil {
					return err
				}
				if t == nil {
					continue
				}
				row.Tag = t
			}
			ret = append(ret, row)
		}
		return nil
	})
	return ret, err
}

func isHexColour(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	for _, c := range s[1:] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return false
		}
	}
	return true
}

var logoTypes = map[string]string{
	"image/png":     ".png",
	"image/jpeg":    ".jpg",
	"image/webp":    ".webp",
	"image/svg+xml": ".svg",
}

// siteLogoFile returns the uploaded logo, if there is one.
func siteLogoFile(cfg *config.Config) (string, bool) {
	base := cfg.SiteLogoPath()
	for _, ext := range logoTypes {
		if _, err := os.Stat(base + ext); err == nil {
			return base + ext, true
		}
	}
	return "", false
}

const maxLogoBytes = 2 << 20

func (r *mutationResolver) SiteSetLogo(ctx context.Context, data string) (*SiteConfig, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	cfg := config.GetInstance()
	base := cfg.SiteLogoPath()
	for _, ext := range logoTypes {
		_ = os.Remove(base + ext)
	}
	if data != "" {
		head, body, ok := strings.Cut(data, ",")
		if !ok || !strings.HasPrefix(head, "data:") || !strings.HasSuffix(head, ";base64") {
			return nil, errors.New("send the picture as a data: address")
		}
		mime := strings.TrimSuffix(strings.TrimPrefix(head, "data:"), ";base64")
		ext, ok := logoTypes[mime]
		if !ok {
			return nil, errors.New("use a PNG, JPEG, WebP or SVG picture")
		}
		b, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			return nil, errors.New("the picture could not be read")
		}
		if len(b) > maxLogoBytes {
			return nil, errors.New("the picture is too big; keep it under 2 MB")
		}
		if err := os.WriteFile(base+ext, b, 0o644); err != nil {
			return nil, err
		}
	}
	return r.siteConfig(ctx)
}

// siteLogoHandler serves the uploaded logo.
func siteLogoHandler(w http.ResponseWriter, r *http.Request) {
	p, ok := siteLogoFile(config.GetInstance())
	if !ok {
		http.NotFound(w, r)
		return
	}
	if strings.HasSuffix(p, ".svg") {
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	http.ServeFile(w, r, p)
}

// ---------------------------------------------------------------- likes, favourites, ratings

func (r *mutationResolver) SiteToggleLike(ctx context.Context, sceneID string) (bool, error) {
	id, err := strconv.Atoi(sceneID)
	if err != nil {
		return false, fmt.Errorf("invalid id %q", sceneID)
	}
	on := false
	err = r.withTxn(ctx, func(ctx context.Context) error {
		visible, err := r.visibleSceneIDs(ctx, []int{id})
		if err != nil {
			return err
		}
		if len(visible) == 0 {
			return errors.New("scene not found")
		}
		on, err = sqlite.SiteToggleFavorite(ctx, sqlite.PlaylistUserID(ctx), sqlite.SiteLikedScene, id)
		return err
	})
	return on, err
}

func (r *mutationResolver) SiteToggleFavoriteModel(ctx context.Context, performerID string) (bool, error) {
	id, err := strconv.Atoi(performerID)
	if err != nil {
		return false, fmt.Errorf("invalid id %q", performerID)
	}
	on := false
	err = r.withTxn(ctx, func(ctx context.Context) error {
		p, err := r.repository.Performer.Find(ctx, id)
		if err != nil {
			return err
		}
		if p == nil {
			return errors.New("model not found")
		}
		on, err = sqlite.SiteToggleFavorite(ctx, sqlite.PlaylistUserID(ctx), sqlite.SiteFavoriteModel, id)
		return err
	})
	return on, err
}

func (r *sceneResolver) SiteLiked(ctx context.Context, obj *models.Scene) (bool, error) {
	var on bool
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		var err error
		on, err = sqlite.SiteIsFavorite(ctx, sqlite.PlaylistUserID(ctx), sqlite.SiteLikedScene, obj.ID)
		return err
	})
	return on, err
}

func (r *sceneResolver) SiteRating(ctx context.Context, obj *models.Scene) (*SiteRating, error) {
	ret := &SiteRating{}
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		avg, n, err := sqlite.SiteRatingSummary(ctx, obj.ID)
		if err != nil {
			return err
		}
		ret.Count = n
		if n > 0 {
			ret.Average = &avg
		}
		return nil
	})
	return ret, err
}

func (r *performerResolver) SiteFavorite(ctx context.Context, obj *models.Performer) (bool, error) {
	var on bool
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		var err error
		on, err = sqlite.SiteIsFavorite(ctx, sqlite.PlaylistUserID(ctx), sqlite.SiteFavoriteModel, obj.ID)
		return err
	})
	return on, err
}

func (r *queryResolver) siteSceneResult(ctx context.Context, fn func(ctx context.Context) ([]int, int, error)) (*SiteSceneResult, error) {
	ret := &SiteSceneResult{Scenes: []*models.Scene{}}
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ids, total, err := fn(ctx)
		if err != nil {
			return err
		}
		ret.Count = total
		ret.Scenes, err = r.scenesInOrder(ctx, ids)
		return err
	})
	return ret, err
}

func (r *queryResolver) SiteLikedScenes(ctx context.Context, page *int, perPage *int) (*SiteSceneResult, error) {
	p, n := sitePage(page, perPage)
	return r.siteSceneResult(ctx, func(ctx context.Context) ([]int, int, error) {
		return sqlite.SiteLikedSceneIDs(ctx, sqlite.PlaylistUserID(ctx), n, (p-1)*n)
	})
}

func (r *queryResolver) SiteFromFavorites(ctx context.Context, page *int, perPage *int) (*SiteSceneResult, error) {
	p, n := sitePage(page, perPage)
	return r.siteSceneResult(ctx, func(ctx context.Context) ([]int, int, error) {
		return sqlite.SiteFromFavoritesIDs(ctx, sqlite.PlaylistUserID(ctx), n, (p-1)*n)
	})
}

func (r *queryResolver) performersInOrder(ctx context.Context, ids []int) ([]*models.Performer, error) {
	if len(ids) == 0 {
		return []*models.Performer{}, nil
	}
	found, err := r.repository.Performer.FindMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[int]*models.Performer, len(found))
	for _, p := range found {
		if p != nil {
			byID[p.ID] = p
		}
	}
	ret := make([]*models.Performer, 0, len(ids))
	for _, id := range ids {
		if p := byID[id]; p != nil {
			ret = append(ret, p)
		}
	}
	return ret, nil
}

func (r *queryResolver) SiteFavoriteModels(ctx context.Context) ([]*models.Performer, error) {
	ret := []*models.Performer{}
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ids, err := sqlite.SiteFavoriteModelIDs(ctx, sqlite.PlaylistUserID(ctx))
		if err != nil {
			return err
		}
		ret, err = r.performersInOrder(ctx, ids)
		return err
	})
	return ret, err
}

func (r *queryResolver) SiteCoStars(ctx context.Context, performerID string, limit *int) ([]*SiteCoStar, error) {
	id, err := strconv.Atoi(performerID)
	if err != nil {
		return nil, fmt.Errorf("invalid id %q", performerID)
	}
	n := 12
	if limit != nil && *limit > 0 && *limit <= 50 {
		n = *limit
	}
	ret := []*SiteCoStar{}
	err = r.withReadTxn(ctx, func(ctx context.Context) error {
		rows, err := sqlite.SiteCoStars(ctx, id, n)
		if err != nil {
			return err
		}
		ids := make([]int, len(rows))
		count := map[int]int{}
		for i, row := range rows {
			ids[i] = row.ID
			count[row.ID] = row.Scenes
		}
		perfs, err := r.performersInOrder(ctx, ids)
		if err != nil {
			return err
		}
		for _, p := range perfs {
			ret = append(ret, &SiteCoStar{Performer: p, Scenes: count[p.ID]})
		}
		return nil
	})
	return ret, err
}
