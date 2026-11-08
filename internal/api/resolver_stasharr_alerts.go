package api

import (
	"context"
	"strconv"
	"time"

	"github.com/stashapp/stash/pkg/sqlite"
)

const alertsMax = 99

func (r *queryResolver) FavoriteAlerts(ctx context.Context, limit *int, since *time.Time) (*FavoriteAlerts, error) {
	n := 20
	if limit != nil && *limit > 0 && *limit <= alertsMax {
		n = *limit
	}
	user := sqlite.PlaylistUserID(ctx)
	ret := &FavoriteAlerts{Alerts: []*FavoriteAlert{}}
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		seen, err := sqlite.AlertsSeenAt(ctx, user)
		if err != nil {
			return err
		}
		ret.SeenAt = seen
		from := seen
		if since != nil && since.After(from) {
			from = *since
		}
		list, err := sqlite.AlertScenes(ctx, user, from, alertsMax+1)
		if err != nil {
			return err
		}
		for _, a := range list {
			// scene lookups follow the viewer's tier
			s, err := r.repository.Scene.Find(ctx, a.SceneID)
			if err != nil {
				return err
			}
			if s == nil {
				continue
			}
			ret.Count++
			if len(ret.Alerts) >= n {
				continue
			}
			added := s.CreatedAt
			if t, ok := sqlite.ParseStatsTime(a.CreatedAt); ok {
				added = t
			}
			ret.Alerts = append(ret.Alerts, &FavoriteAlert{
				Scene:       s,
				PerformerID: strconv.Itoa(a.PerformerID),
				Performer:   a.Performer,
				AddedAt:     added,
			})
		}
		if ret.Count > alertsMax {
			ret.Count = alertsMax
		}
		return nil
	})
	return ret, err
}

func (r *mutationResolver) MarkFavoriteAlertsSeen(ctx context.Context) (bool, error) {
	user := sqlite.PlaylistUserID(ctx)
	err := r.withTxn(ctx, func(ctx context.Context) error {
		return sqlite.MarkAlertsSeen(ctx, user)
	})
	return err == nil, err
}
