package sqlite

// Stasharr: recommendations by taste. Every indexed scene has a mean
// vector of what it looks like; a person's taste is the mean of the scenes
// they played, rated or liked, and scenes that look like it rank higher.

import (
	"context"
	"math"
	"sync"
	"time"
)

func dequantizeVec(vec []byte, scale float64) []float32 {
	out := make([]float32, len(vec))
	for i, b := range vec {
		out[i] = float32(float64(int8(b)) * scale)
	}
	return out
}

func normalise(v []float32) []float32 {
	var n float64
	for _, x := range v {
		n += float64(x) * float64(x)
	}
	if n == 0 {
		return v
	}
	s := float32(1 / math.Sqrt(n))
	for i := range v {
		v[i] *= s
	}
	return v
}

func quantizeVec(v []float32) ([]byte, float64) {
	maxAbs := 0.0
	for _, x := range v {
		maxAbs = math.Max(maxAbs, math.Abs(float64(x)))
	}
	if maxAbs == 0 {
		maxAbs = 1
	}
	scale := maxAbs / 127
	out := make([]byte, len(v))
	for i, x := range v {
		out[i] = byte(int8(math.Round(float64(x) / scale)))
	}
	return out, scale
}

// meanOf returns the normalised mean of frame vectors.
func meanOf(list []FrameVector) []float32 {
	if len(list) == 0 {
		return nil
	}
	var sum []float32
	for _, f := range list {
		v := normalise(dequantizeVec(f.Vec, f.Scale))
		if sum == nil {
			sum = make([]float32, len(v))
		}
		if len(v) != len(sum) {
			continue
		}
		for i, x := range v {
			sum[i] += x
		}
	}
	return normalise(sum)
}

func saveSceneMean(ctx context.Context, sceneID int, model string, list []FrameVector) error {
	m := meanOf(list)
	if m == nil {
		_, err := dbWrapper.Exec(ctx, "DELETE FROM scene_mean_vectors WHERE scene_id = ?", sceneID)
		return err
	}
	q, scale := quantizeVec(m)
	_, err := dbWrapper.Exec(ctx, `INSERT INTO scene_mean_vectors (scene_id, model, scale, vec) VALUES (?, ?, ?, ?)
ON CONFLICT(scene_id) DO UPDATE SET model = excluded.model, scale = excluded.scale, vec = excluded.vec`, sceneID, model, scale, q)
	forgetSceneMeans()
	return err
}

// BackfillSceneMeans makes mean vectors for indexed scenes that have none
// (indexed before this version), at most limit of them. It returns how many
// it made.
func BackfillSceneMeans(ctx context.Context, limit int) (int, error) {
	type run struct {
		SceneID int    `db:"scene_id"`
		Model   string `db:"model"`
	}
	var runs []run
	if err := dbWrapper.Select(ctx, &runs, `SELECT r.scene_id, r.model FROM scene_vector_runs r
WHERE r.frames > 0 AND NOT EXISTS (SELECT 1 FROM scene_mean_vectors m WHERE m.scene_id = r.scene_id AND m.model = r.model)
LIMIT ?`, limit); err != nil {
		return 0, err
	}
	for _, r := range runs {
		var list []FrameVector
		if err := dbWrapper.Select(ctx, &list, "SELECT scene_id, seconds, scale, vec FROM scene_frame_vectors WHERE scene_id = ?", r.SceneID); err != nil {
			return 0, err
		}
		if err := saveSceneMean(ctx, r.SceneID, r.Model, list); err != nil {
			return 0, err
		}
	}
	return len(runs), nil
}

// ---------------------------------------------------------------- the cache

type sceneMeans struct {
	model  string
	vecs   map[int][]float32
	loaded time.Time
}

var meansCache struct {
	mu sync.Mutex
	m  *sceneMeans
}

func forgetSceneMeans() {
	meansCache.mu.Lock()
	meansCache.m = nil
	meansCache.mu.Unlock()
}

// loadSceneMeans returns every scene's mean vector for the model most
// scenes were indexed with, cached for 15 minutes.
func loadSceneMeans(ctx context.Context) (map[int][]float32, error) {
	meansCache.mu.Lock()
	defer meansCache.mu.Unlock()
	if m := meansCache.m; m != nil && time.Since(m.loaded) < 15*time.Minute {
		return m.vecs, nil
	}
	var models []string
	if err := dbWrapper.Select(ctx, &models, "SELECT model FROM scene_mean_vectors GROUP BY model ORDER BY COUNT(*) DESC LIMIT 1"); err != nil {
		return nil, err
	}
	if len(models) == 0 {
		meansCache.m = &sceneMeans{vecs: map[int][]float32{}, loaded: time.Now()}
		return meansCache.m.vecs, nil
	}
	type row struct {
		SceneID int     `db:"scene_id"`
		Scale   float64 `db:"scale"`
		Vec     []byte  `db:"vec"`
	}
	var rows []row
	if err := dbWrapper.Select(ctx, &rows, "SELECT scene_id, scale, vec FROM scene_mean_vectors WHERE model = ?", models[0]); err != nil {
		return nil, err
	}
	vecs := make(map[int][]float32, len(rows))
	for _, r := range rows {
		vecs[r.SceneID] = normalise(dequantizeVec(r.Vec, r.Scale))
	}
	meansCache.m = &sceneMeans{model: models[0], vecs: vecs, loaded: time.Now()}
	return vecs, nil
}

// tasteWeight is how much looks count against performers, tags and studios.
const tasteWeight = 0.45

// blendTaste adds how much each scene looks like what the person likes to
// the tag-based scores. seeds are the person's scenes with their weights;
// scores is changed in place.
func blendTaste(ctx context.Context, seeds map[int]float64, watched map[int]bool, scores map[int]float64) {
	means, err := loadSceneMeans(ctx)
	if err != nil || len(means) < 20 {
		return
	}
	var taste []float32
	used := 0
	for id, w := range seeds {
		v, ok := means[id]
		if !ok || w <= 0 {
			continue
		}
		if taste == nil {
			taste = make([]float32, len(v))
		}
		if len(v) != len(taste) {
			continue
		}
		for i, x := range v {
			taste[i] += float32(w) * x
		}
		used++
	}
	if used == 0 {
		return
	}
	taste = normalise(taste)

	sims := make(map[int]float64, len(means))
	var sum, sq float64
	for id, v := range means {
		if watched[id] || len(v) != len(taste) {
			continue
		}
		var dot float32
		for i, x := range v {
			dot += x * taste[i]
		}
		s := float64(dot)
		sims[id] = s
		sum += s
		sq += s * s
	}
	n := float64(len(sims))
	if n < 10 {
		return
	}
	mean := sum / n
	std := math.Sqrt(math.Max(sq/n-mean*mean, 1e-9))

	// tag scores to 0..1
	maxTag := 0.0
	for _, s := range scores {
		maxTag = math.Max(maxTag, s)
	}
	for id, s := range scores {
		if maxTag > 0 {
			scores[id] = (1 - tasteWeight) * s / maxTag
		}
	}
	for id, s := range sims {
		z := (s - mean) / std
		if z <= 0 {
			continue
		}
		scores[id] += tasteWeight * math.Min(z, 3) / 3
	}
}
