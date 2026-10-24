package sqlite

// Stasharr recommendations, worked out from the metadata already in the
// database: shared performers, tags and studios. Tags that many scenes share
// count for less than rare ones. No AI or extra service is needed.
//
// The functions return more ids than needed, best first; the caller loads
// them through the normal scene queries, which drop scenes the current
// user's tier hides.

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"

	"github.com/stashapp/stash/pkg/access"
)

const (
	recPerformerWeight = 4.0
	recStudioWeight    = 3.0
	recNetworkWeight   = 1.5
	// tags on more than this share of scenes say little about a scene
	recCommonTagShare = 0.4
	// how much of the user's history is used
	recHistoryScenes = 300
)

type idWeight struct {
	ID     int     `db:"id"`
	Weight float64 `db:"weight"`
}

func intArgs(ids []int) []interface{} {
	ret := make([]interface{}, len(ids))
	for i, id := range ids {
		ret[i] = id
	}
	return ret
}

func placeholders(n int) string {
	if n <= 0 {
		return "NULL"
	}
	b := make([]byte, 0, n*2)
	for i := 0; i < n; i++ {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, '?')
	}
	return string(b)
}

func sceneTotal(ctx context.Context) (int, error) {
	var n int
	err := dbWrapper.Get(ctx, &n, "SELECT COUNT(*) FROM scenes")
	return n, err
}

// tagWeights returns an idf style weight for each tag: rare tags count more.
// Very common tags are left out.
func tagWeights(ctx context.Context, tagIDs []int, total int) (map[int]float64, error) {
	ret := map[int]float64{}
	if len(tagIDs) == 0 || total == 0 {
		return ret, nil
	}
	var rows []idWeight
	q := fmt.Sprintf("SELECT tag_id AS id, COUNT(*) AS weight FROM scenes_tags WHERE tag_id IN (%s) GROUP BY tag_id", placeholders(len(tagIDs)))
	if err := dbWrapper.Select(ctx, &rows, q, intArgs(tagIDs)...); err != nil {
		return nil, err
	}
	for _, r := range rows {
		df := r.Weight
		if df/float64(total) > recCommonTagShare && total > 20 {
			continue
		}
		ret[r.ID] = math.Max(math.Log(float64(total)/(1+df)), 0.2)
	}
	return ret, nil
}

// profile holds how much a set of scenes is about each performer, tag and
// studio.
type profile struct {
	performers map[int]float64
	tags       map[int]float64
	studios    map[int]float64
	networks   map[int]float64
}

func newProfile() *profile {
	return &profile{
		performers: map[int]float64{},
		tags:       map[int]float64{},
		studios:    map[int]float64{},
		networks:   map[int]float64{},
	}
}

func (p *profile) empty() bool {
	return len(p.performers) == 0 && len(p.tags) == 0 && len(p.studios) == 0
}

// build fills the profile from seed scenes and how much each one counts.
func (p *profile) build(ctx context.Context, seeds map[int]float64) error {
	if len(seeds) == 0 {
		return nil
	}
	ids := make([]int, 0, len(seeds))
	for id := range seeds {
		ids = append(ids, id)
	}

	return batchExec(ids, defaultBatchSize, func(batch []int) error {
		in := placeholders(len(batch))
		args := intArgs(batch)

		type pair struct {
			SceneID int           `db:"scene_id"`
			OtherID sql.NullInt64 `db:"other_id"`
		}
		add := func(q string, into map[int]float64) error {
			var rows []pair
			if err := dbWrapper.Select(ctx, &rows, q, args...); err != nil {
				return err
			}
			for _, r := range rows {
				if r.OtherID.Valid {
					into[int(r.OtherID.Int64)] += seeds[r.SceneID]
				}
			}
			return nil
		}
		if err := add("SELECT scene_id, performer_id AS other_id FROM performers_scenes WHERE scene_id IN ("+in+")", p.performers); err != nil {
			return err
		}
		if err := add("SELECT scene_id, tag_id AS other_id FROM scenes_tags WHERE scene_id IN ("+in+")", p.tags); err != nil {
			return err
		}
		if err := add("SELECT id AS scene_id, studio_id AS other_id FROM scenes WHERE id IN ("+in+")", p.studios); err != nil {
			return err
		}
		return add("SELECT scenes.id AS scene_id, studios.parent_id AS other_id FROM scenes JOIN studios ON studios.id = scenes.studio_id WHERE scenes.id IN ("+in+")", p.networks)
	})
}

// top returns the n highest weighted positive ids.
func top(m map[int]float64, n int) map[int]float64 {
	type kv struct {
		id int
		w  float64
	}
	list := make([]kv, 0, len(m))
	for id, w := range m {
		if w > 0 {
			list = append(list, kv{id, w})
		}
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].w != list[j].w {
			return list[i].w > list[j].w
		}
		return list[i].id < list[j].id
	})
	if len(list) > n {
		list = list[:n]
	}
	ret := make(map[int]float64, len(list))
	for _, e := range list {
		ret[e.id] = e.w
	}
	return ret
}

func keys(m map[int]float64) []int {
	ret := make([]int, 0, len(m))
	for k := range m {
		ret = append(ret, k)
	}
	sort.Ints(ret)
	return ret
}

// score rates every scene that shares something with the profile.
func (p *profile) score(ctx context.Context, normalise bool) (map[int]float64, error) {
	scores := map[int]float64{}
	total, err := sceneTotal(ctx)
	if err != nil {
		return nil, err
	}

	performers := top(p.performers, 40)
	tags := top(p.tags, 60)
	studios := top(p.studios, 20)
	networks := top(p.networks, 10)

	// scale profile weights so the strongest of each kind counts 1
	scale := func(m map[int]float64) {
		if !normalise {
			return
		}
		maxW := 0.0
		for _, w := range m {
			maxW = math.Max(maxW, w)
		}
		if maxW == 0 {
			return
		}
		for k, w := range m {
			m[k] = w / maxW
		}
	}
	scale(performers)
	scale(tags)
	scale(studios)
	scale(networks)

	// each kind is matched with one query; the per-row weight is looked up
	// afterwards, as sqlite cannot take a map
	type match struct {
		SceneID int `db:"scene_id"`
		OtherID int `db:"other_id"`
	}
	apply := func(q string, ids []int, weight func(id int) float64) error {
		if len(ids) == 0 {
			return nil
		}
		var rows []match
		if err := dbWrapper.Select(ctx, &rows, fmt.Sprintf(q, placeholders(len(ids))), intArgs(ids)...); err != nil {
			return err
		}
		for _, r := range rows {
			scores[r.SceneID] += weight(r.OtherID)
		}
		return nil
	}

	if err := apply("SELECT scene_id, performer_id AS other_id FROM performers_scenes WHERE performer_id IN (%s)", keys(performers),
		func(id int) float64 { return recPerformerWeight * performers[id] }); err != nil {
		return nil, err
	}

	idf, err := tagWeights(ctx, keys(tags), total)
	if err != nil {
		return nil, err
	}
	if err := apply("SELECT scene_id, tag_id AS other_id FROM scenes_tags WHERE tag_id IN (%s)", keys(idf),
		func(id int) float64 { return idf[id] * tags[id] }); err != nil {
		return nil, err
	}

	if err := apply("SELECT id AS scene_id, studio_id AS other_id FROM scenes WHERE studio_id IN (%s)", keys(studios),
		func(id int) float64 { return recStudioWeight * studios[id] }); err != nil {
		return nil, err
	}
	if err := apply("SELECT scenes.id AS scene_id, studios.parent_id AS other_id FROM scenes JOIN studios ON studios.id = scenes.studio_id WHERE studios.parent_id IN (%s)", keys(networks),
		func(id int) float64 { return recNetworkWeight * networks[id] }); err != nil {
		return nil, err
	}
	return scores, nil
}

func ranked(scores map[int]float64, exclude map[int]bool, limit int) []int {
	type kv struct {
		id int
		w  float64
	}
	list := make([]kv, 0, len(scores))
	for id, w := range scores {
		if w > 0 && !exclude[id] {
			list = append(list, kv{id, w})
		}
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].w != list[j].w {
			return list[i].w > list[j].w
		}
		return list[i].id > list[j].id
	})
	if len(list) > limit {
		list = list[:limit]
	}
	ret := make([]int, len(list))
	for i, e := range list {
		ret[i] = e.id
	}
	return ret
}

// SimilarSceneIDs returns scenes like the given one, best first.
func SimilarSceneIDs(ctx context.Context, sceneID int, limit int) ([]int, error) {
	p := newProfile()
	if err := p.build(ctx, map[int]float64{sceneID: 1}); err != nil {
		return nil, err
	}
	if p.empty() {
		return nil, nil
	}
	scores, err := p.score(ctx, false)
	if err != nil {
		return nil, err
	}
	return ranked(scores, map[int]bool{sceneID: true}, limit), nil
}

// RecommendedSceneIDs returns scenes the current user has not watched that
// match what they watch, rate highly and O, best first. It returns nothing
// when the user has no history yet.
func RecommendedSceneIDs(ctx context.Context, limit int) ([]int, error) {
	uid := access.HistoryUserID(ctx)
	userCond := "user_id IS NULL"
	var userArgs []interface{}
	if uid != nil {
		userCond = "user_id = ?"
		userArgs = []interface{}{*uid}
	}

	seeds := map[int]float64{}
	watched := map[int]bool{}

	// recent plays
	var plays []idWeight
	q := fmt.Sprintf(`SELECT scene_id AS id, COUNT(*) AS weight FROM scenes_view_dates WHERE %s
GROUP BY scene_id ORDER BY MAX(view_date) DESC LIMIT %d`, userCond, recHistoryScenes)
	if err := dbWrapper.Select(ctx, &plays, q, userArgs...); err != nil {
		return nil, err
	}
	for _, r := range plays {
		watched[r.ID] = true
		seeds[r.ID] += math.Min(r.Weight, 3)
	}

	// O history counts double
	var oRows []idWeight
	q = fmt.Sprintf(`SELECT scene_id AS id, COUNT(*) AS weight FROM scenes_o_dates WHERE %s
GROUP BY scene_id ORDER BY MAX(o_date) DESC LIMIT %d`, userCond, recHistoryScenes)
	if err := dbWrapper.Select(ctx, &oRows, q, userArgs...); err != nil {
		return nil, err
	}
	for _, r := range oRows {
		watched[r.ID] = true
		seeds[r.ID] += 2 * math.Min(r.Weight, 3)
	}

	// ratings: above 3 stars pull towards, below push away
	var ratings []idWeight
	if uid != nil {
		q = fmt.Sprintf("SELECT scene_id AS id, rating AS weight FROM users_scenes WHERE user_id = ? AND rating IS NOT NULL LIMIT %d", recHistoryScenes)
		if err := dbWrapper.Select(ctx, &ratings, q, *uid); err != nil {
			return nil, err
		}
	} else {
		q = fmt.Sprintf("SELECT id, rating AS weight FROM scenes WHERE rating IS NOT NULL ORDER BY updated_at DESC LIMIT %d", recHistoryScenes)
		if err := dbWrapper.Select(ctx, &ratings, q); err != nil {
			return nil, err
		}
	}
	for _, r := range ratings {
		watched[r.ID] = true
		seeds[r.ID] += (r.Weight - 60) / 20
	}

	p := newProfile()
	if err := p.build(ctx, seeds); err != nil {
		return nil, err
	}
	if p.empty() {
		return nil, nil
	}
	scores, err := p.score(ctx, true)
	if err != nil {
		return nil, err
	}
	return ranked(scores, watched, limit), nil
}
