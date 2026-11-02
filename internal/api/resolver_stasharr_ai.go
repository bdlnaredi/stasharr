package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stashapp/stash/pkg/sqlite"
)

func toAILabels(in []manager.AILabel) []*AIMarkerLabel {
	ret := make([]*AIMarkerLabel, len(in))
	for i, l := range in {
		ret[i] = &AIMarkerLabel{Name: l.Name, Prompt: l.Prompt, Enabled: l.Enabled, Marker: l.Marker}
	}
	return ret
}

func aiMarkerConfig() *AIMarkerConfig {
	cfg := config.GetInstance()
	return &AIMarkerConfig{
		URL:           cfg.GetAIURL(),
		Interval:      cfg.GetAIInterval(),
		MinConfidence: cfg.GetAIMinConfidence(),
		MinSeconds:    cfg.GetAIMinSeconds(),
		Auto:          cfg.GetAIAuto(),
		Review:        cfg.GetAIReview(),
		Labels:        toAILabels(manager.AILabels(cfg)),
		DefaultLabels: toAILabels(manager.DefaultAILabels),
	}
}

func (r *queryResolver) AiMarkerConfig(ctx context.Context) (*AIMarkerConfig, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	return aiMarkerConfig(), nil
}

func (r *mutationResolver) ConfigureAIMarkers(ctx context.Context, input AIMarkerConfigInput) (*AIMarkerConfig, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	cfg := config.GetInstance()
	if input.URL != nil {
		u := strings.TrimRight(strings.TrimSpace(*input.URL), "/")
		if u != "" {
			p, err := url.Parse(u)
			if err != nil || (p.Scheme != "http" && p.Scheme != "https") || p.Host == "" {
				return nil, errors.New("the address must look like http://stasharr-tagger:2705")
			}
		}
		cfg.SetString(config.AIURL, u)
	}
	if input.Interval != nil {
		if *input.Interval < 1 || *input.Interval > 30 {
			return nil, errors.New("look every 1 to 30 seconds")
		}
		cfg.SetFloat(config.AIInterval, *input.Interval)
	}
	if input.MinConfidence != nil {
		if *input.MinConfidence <= 0 || *input.MinConfidence >= 1 {
			return nil, errors.New("confidence must be between 0 and 1")
		}
		cfg.SetFloat(config.AIMinConfidence, *input.MinConfidence)
	}
	if input.MinSeconds != nil {
		if *input.MinSeconds < 3 || *input.MinSeconds > 600 {
			return nil, errors.New("shortest marker must be 3 to 600 seconds")
		}
		cfg.SetFloat(config.AIMinSeconds, *input.MinSeconds)
	}
	if input.Auto != nil {
		cfg.SetBool(config.AIAuto, *input.Auto)
	}
	if input.Review != nil {
		cfg.SetBool(config.AIReview, *input.Review)
	}
	if input.Labels != nil {
		if len(input.Labels) == 0 {
			cfg.SetString(config.AILabels, "")
		} else {
			var labels []manager.AILabel
			seen := map[string]bool{}
			for _, l := range input.Labels {
				if l == nil {
					continue
				}
				name := strings.TrimSpace(l.Name)
				prompt := strings.TrimSpace(l.Prompt)
				if name == "" || prompt == "" {
					return nil, errors.New("every label needs a name and a description")
				}
				if seen[strings.ToLower(name)] {
					return nil, fmt.Errorf("the label %q is listed twice", name)
				}
				seen[strings.ToLower(name)] = true
				labels = append(labels, manager.AILabel{Name: name, Prompt: prompt, Enabled: l.Enabled, Marker: l.Marker})
			}
			b, err := json.Marshal(labels)
			if err != nil {
				return nil, err
			}
			cfg.SetString(config.AILabels, string(b))
		}
	}
	if err := cfg.Write(); err != nil {
		return nil, err
	}
	return aiMarkerConfig(), nil
}

func (r *queryResolver) AiTaggerStatus(ctx context.Context) (*AITaggerStatus, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	ret := &AITaggerStatus{}
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		var err error
		ret.Done, ret.Total, err = sqlite.AIProgress(ctx)
		return err
	}); err != nil {
		return nil, err
	}
	u := config.GetInstance().GetAIURL()
	if u == "" {
		msg := "No tagger address set"
		ret.Error = &msg
		return ret, nil
	}
	st, err := manager.GetTaggerStatus(ctx, u)
	if err != nil {
		msg := err.Error()
		ret.Error = &msg
		return ret, nil
	}
	ret.Reachable = true
	ret.Ok = st.OK
	ret.Loading = st.Loading
	ret.Error = st.Error
	ret.Gpu = st.GPU
	if st.Device != "" {
		ret.Device = &st.Device
	}
	if st.Model != "" {
		ret.Model = &st.Model
	}
	return ret, nil
}

func (r *queryResolver) AiMarkerSuggestions(ctx context.Context, page *int, perPage *int) (*AIMarkerSuggestionResult, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return nil, err
	}
	p, pp := 1, 100
	if page != nil && *page > 0 {
		p = *page
	}
	if perPage != nil && *perPage > 0 && *perPage <= 500 {
		pp = *perPage
	}
	ret := &AIMarkerSuggestionResult{Suggestions: []*AIMarkerSuggestion{}}
	var list []*sqlite.AIMarkerSuggestion
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		var err error
		list, ret.Count, ret.Scenes, err = sqlite.AISuggestions(ctx, pp, (p-1)*pp)
		return err
	}); err != nil {
		return nil, err
	}
	for _, s := range list {
		scene, err := loadersSceneByID(ctx, s.SceneID)
		if err != nil {
			return nil, err
		}
		if scene == nil {
			continue
		}
		ret.Suggestions = append(ret.Suggestions, &AIMarkerSuggestion{
			ID:         strconv.Itoa(s.ID),
			Scene:      scene,
			Label:      s.Label,
			Seconds:    s.Seconds,
			EndSeconds: s.EndSeconds,
			Score:      s.Score,
		})
	}
	return ret, nil
}

func (r *mutationResolver) RunAIMarkers(ctx context.Context, sceneIds []string) (string, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return "", err
	}
	if config.GetInstance().GetAIURL() == "" {
		return "", errors.New("set the Stasharr Tagger address first")
	}
	ids, err := stringslice.StringSliceToIntSlice(sceneIds)
	if err != nil {
		return "", err
	}
	name := "AI markers: every scene not looked at yet"
	if len(ids) > 0 {
		name = fmt.Sprintf("AI markers: %d scene(s)", len(ids))
	}
	// jobs run without the requesting user, so they see the whole library
	jobID := manager.GetInstance().JobManager.Add(context.WithoutCancel(access.WithPrincipal(ctx, nil)), name, &manager.AIMarkerJob{SceneIDs: ids})
	return strconv.Itoa(jobID), nil
}

func (r *mutationResolver) AiMarkerApprove(ctx context.Context, ids []string) (int, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return 0, err
	}
	intIDs, err := stringslice.StringSliceToIntSlice(ids)
	if err != nil {
		return 0, err
	}
	made := 0
	err = r.withTxn(ctx, func(ctx context.Context) error {
		list, err := sqlite.AIFindSuggestions(ctx, intIDs)
		if err != nil {
			return err
		}
		for _, s := range list {
			ok, err := manager.SaveAIMarker(ctx, r.repository, s.SceneID, s.Label, s.Seconds, s.EndSeconds)
			if err != nil {
				return err
			}
			if ok {
				made++
			}
			if err := sqlite.AIDeleteSuggestion(ctx, s.ID); err != nil {
				return err
			}
		}
		return nil
	})
	return made, err
}

func (r *mutationResolver) AiMarkerReject(ctx context.Context, ids []string) (int, error) {
	if err := access.RequireRole(ctx, access.RoleAdmin); err != nil {
		return 0, err
	}
	intIDs, err := stringslice.StringSliceToIntSlice(ids)
	if err != nil {
		return 0, err
	}
	n := 0
	err = r.withTxn(ctx, func(ctx context.Context) error {
		for _, id := range intIDs {
			if err := sqlite.AIDeleteSuggestion(ctx, id); err != nil {
				return err
			}
			n++
		}
		return nil
	})
	return n, err
}
