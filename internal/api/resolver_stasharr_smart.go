package api

// Smart playlists: playlists whose scenes come from a saved scene filter.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/stashapp/stash/pkg/models"
)

const (
	smartDefaultLimit = 100
	smartMaxLimit     = 1000
)

var errSmartPlaylist = errors.New("smart playlists fill themselves from their filter; edit the filter instead")

// smartRuleJSON checks a rule from the API and returns it as stored JSON.
func smartRuleJSON(in *SmartPlaylistRuleInput) (string, error) {
	rule := models.SmartRule{Filter: in.SceneFilter, Limit: smartDefaultLimit}
	if in.Sort != nil {
		rule.Sort = *in.Sort
	}
	if in.Direction != nil {
		rule.Direction = string(*in.Direction)
	}
	if in.Limit != nil && *in.Limit > 0 {
		rule.Limit = *in.Limit
	}
	if rule.Limit > smartMaxLimit {
		rule.Limit = smartMaxLimit
	}
	if rule.Filter == nil {
		rule.Filter = &models.SceneFilterType{}
	}
	b, err := json.Marshal(rule)
	if err != nil {
		return "", fmt.Errorf("saving the filter: %w", err)
	}
	return string(b), nil
}

func parseSmartRule(p *models.Playlist) (*models.SmartRule, error) {
	var rule models.SmartRule
	if err := json.Unmarshal([]byte(p.RuleJSON), &rule); err != nil {
		return nil, fmt.Errorf("playlist %s has an unreadable filter: %w", p.Name, err)
	}
	if rule.Limit <= 0 {
		rule.Limit = smartDefaultLimit
	}
	return &rule, nil
}

// smartQuery runs a smart playlist's filter for the current user, so their
// tier still applies. Must run in a transaction.
func (r *Resolver) smartQuery(ctx context.Context, p *models.Playlist, limit int, count bool) (*models.SceneQueryResult, int, error) {
	rule, err := parseSmartRule(p)
	if err != nil {
		return nil, 0, err
	}
	if limit <= 0 || limit > rule.Limit {
		limit = rule.Limit
	}
	ff := &models.FindFilterType{PerPage: &limit}
	if rule.Sort != "" {
		sort := rule.Sort
		ff.Sort = &sort
	}
	if rule.Direction != "" {
		dir := models.SortDirectionEnum(rule.Direction)
		ff.Direction = &dir
	}
	res, err := r.repository.Scene.Query(ctx, models.SceneQueryOptions{
		QueryOptions: models.QueryOptions{FindFilter: ff, Count: count},
		SceneFilter:  rule.Filter,
	})
	if err != nil {
		return nil, 0, err
	}
	return res, rule.Limit, nil
}

// playlistSceneIDs returns the scenes of any playlist, smart or not. Must run
// in a transaction.
func (r *Resolver) playlistSceneIDs(ctx context.Context, p *models.Playlist, limit int) ([]int, error) {
	if !p.IsSmart() {
		ids, err := r.repository.Playlist.SceneIDs(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		if limit > 0 && len(ids) > limit {
			ids = ids[:limit]
		}
		return ids, nil
	}
	res, _, err := r.smartQuery(ctx, p, limit, false)
	if err != nil {
		return nil, err
	}
	return res.IDs, nil
}

func (r *Resolver) playlistSceneCount(ctx context.Context, p *models.Playlist) (int, error) {
	if !p.IsSmart() {
		return r.repository.Playlist.SceneCount(ctx, p.ID)
	}
	one := 1
	res, limit, err := r.smartQuery(ctx, p, one, true)
	if err != nil {
		return 0, err
	}
	if res.Count > limit {
		return limit, nil
	}
	return res.Count, nil
}

func (r *playlistResolver) Smart(ctx context.Context, obj *models.Playlist) (bool, error) {
	return obj.IsSmart(), nil
}

func (r *playlistResolver) SmartRule(ctx context.Context, obj *models.Playlist) (*SmartPlaylistRule, error) {
	if !obj.IsSmart() {
		return nil, nil
	}
	rule, err := parseSmartRule(obj)
	if err != nil {
		return nil, err
	}
	filter := "{}"
	if rule.Filter != nil {
		b, err := json.Marshal(rule.Filter)
		if err != nil {
			return nil, err
		}
		filter = string(b)
	}
	ret := &SmartPlaylistRule{Filter: filter, Limit: rule.Limit}
	if rule.Sort != "" {
		s := rule.Sort
		ret.Sort = &s
	}
	if rule.Direction != "" {
		d := models.SortDirectionEnum(rule.Direction)
		ret.Direction = &d
	}
	return ret, nil
}
