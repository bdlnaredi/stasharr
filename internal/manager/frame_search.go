package manager

// Search by description: every scene gets a small vector for one frame
// every 10 seconds or so, made by the Stasharr Tagger. A typed description
// becomes a vector the same way, and the frames closest to it are the
// matching moments.

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sqlite"
)

const (
	searchFrameEvery = 10.0
	searchMaxFrames  = 150
)

type embedResponse struct {
	Vectors [][]float64 `json:"vectors"`
	Dim     int         `json:"dim"`
	Model   string      `json:"model"`
	Detail  string      `json:"detail"`
}

func taggerPost(ctx context.Context, url, path string, payload any) (*embedResponse, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url+path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := taggerClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tagger: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 64<<20))
	out := &embedResponse{}
	_ = json.Unmarshal(body, out)
	if res.StatusCode == http.StatusNotFound {
		return nil, errors.New("this Stasharr Tagger is too old for search by description; update its image")
	}
	if res.StatusCode != http.StatusOK {
		msg := out.Detail
		if msg == "" {
			msg = strings.TrimSpace(string(body))
		}
		return nil, fmt.Errorf("tagger: %s: %s", res.Status, msg)
	}
	return out, nil
}

// quantize stores a vector as signed bytes and a scale.
func quantize(v []float64) ([]byte, float64) {
	maxAbs := 0.0
	for _, x := range v {
		maxAbs = math.Max(maxAbs, math.Abs(x))
	}
	if maxAbs == 0 {
		maxAbs = 1
	}
	scale := maxAbs / 127
	out := make([]byte, len(v))
	for i, x := range v {
		out[i] = byte(int8(math.Round(x / scale)))
	}
	return out, scale
}

// ---------------------------------------------------------------- the index job

// SearchIndexJob makes search vectors for scenes.
type SearchIndexJob struct {
	// SceneIDs to index; empty for every scene not indexed with the
	// tagger's current model.
	SceneIDs []int
}

func (j *SearchIndexJob) Execute(ctx context.Context, progress *job.Progress) error {
	mgr := GetInstance()
	url := mgr.Config.GetAIURL()
	if url == "" {
		return errors.New("set the Stasharr Tagger address first (Settings → AI markers)")
	}
	st, err := GetTaggerStatus(ctx, url)
	if err != nil {
		return err
	}
	if !st.OK {
		return errors.New("the Stasharr Tagger is not ready yet")
	}
	model := st.Model

	ids := j.SceneIDs
	if len(ids) == 0 {
		if err := mgr.Repository.WithReadTxn(ctx, func(ctx context.Context) error {
			var err error
			ids, err = sqlite.SceneIDsToIndex(ctx, model)
			return err
		}); err != nil {
			return err
		}
	}

	progress.SetTotal(len(ids))
	done := 0
	defer func() {
		if done > 0 {
			frameIndexCache.forget()
		}
	}()
	for _, id := range ids {
		if job.IsCancelled(ctx) {
			return nil
		}
		progress.ExecuteTask(fmt.Sprintf("Search index: scene %d", id), func() {
			if err := indexScene(ctx, url, model, id); err != nil {
				if !errors.Is(err, context.Canceled) {
					logger.Warnf("[search] scene %d: %v", id, err)
				}
				return
			}
			done++
			// let searches see new scenes every so often
			if done%25 == 0 {
				frameIndexCache.forget()
			}
		})
		progress.Increment()
	}
	logger.Infof("[search] indexed %d scene(s)", done)
	return nil
}

func indexScene(ctx context.Context, url, model string, sceneID int) error {
	r := GetInstance().Repository
	var vf *models.VideoFile
	if err := r.WithReadTxn(ctx, func(ctx context.Context) error {
		s, err := r.Scene.Find(ctx, sceneID)
		if err != nil || s == nil {
			return err
		}
		if err := s.LoadPrimaryFile(ctx, r.File); err != nil {
			return err
		}
		vf = s.Files.Primary()
		return nil
	}); err != nil {
		return err
	}

	var list []sqlite.FrameVector
	if vf != nil && !vf.IsAudioOnly() && vf.Duration > 0 {
		interval := math.Max(searchFrameEvery, vf.Duration/searchMaxFrames)
		frames, err := sampleFrames(ctx, vf.Path, interval)
		if err != nil {
			return err
		}
		for i := 0; i < len(frames); i += 32 {
			end := min(i+32, len(frames))
			imgs := make([]string, 0, end-i)
			for _, f := range frames[i:end] {
				imgs = append(imgs, base64.StdEncoding.EncodeToString(f))
			}
			res, err := taggerPost(ctx, url, "/embed_images", map[string]any{"images": imgs})
			if err != nil {
				return err
			}
			for k, v := range res.Vectors {
				q, scale := quantize(v)
				list = append(list, sqlite.FrameVector{SceneID: sceneID, Seconds: round2(float64(i+k) * interval), Scale: scale, Vec: q})
			}
		}
	}
	return r.WithTxn(ctx, func(ctx context.Context) error {
		return sqlite.SaveFrameVectors(ctx, sceneID, model, list)
	})
}

// ---------------------------------------------------------------- the index in memory

type frameIndex struct {
	model  string
	dim    int
	scenes []int32
	secs   []float32
	scales []float32
	vecs   []int8
}

type frameIndexHolder struct {
	mu    sync.Mutex
	index *frameIndex
}

var frameIndexCache frameIndexHolder

func (h *frameIndexHolder) forget() {
	h.mu.Lock()
	h.index = nil
	h.mu.Unlock()
}

func (h *frameIndexHolder) get(ctx context.Context, model string) (*frameIndex, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.index != nil && h.index.model == model {
		return h.index, nil
	}
	idx := &frameIndex{model: model}
	r := GetInstance().Repository
	started := time.Now()
	err := r.WithReadTxn(ctx, func(ctx context.Context) error {
		return sqlite.AllFrameVectors(ctx, model, func(v sqlite.FrameVector) {
			if idx.dim == 0 {
				idx.dim = len(v.Vec)
			}
			if len(v.Vec) != idx.dim {
				return
			}
			idx.scenes = append(idx.scenes, int32(v.SceneID))
			idx.secs = append(idx.secs, float32(v.Seconds))
			idx.scales = append(idx.scales, float32(v.Scale))
			for _, b := range v.Vec {
				idx.vecs = append(idx.vecs, int8(b))
			}
		})
	})
	if err != nil {
		return nil, err
	}
	logger.Debugf("[search] loaded %d frames in %s", len(idx.scenes), time.Since(started).Round(time.Millisecond))
	h.index = idx
	return idx, nil
}

// FrameHit is a matching moment.
type FrameHit struct {
	SceneID int
	Seconds float64
	Score   float64
}

// SearchFrames finds the moments that look most like the description, one
// per scene, best first. It returns up to limit scenes.
func SearchFrames(ctx context.Context, text string, limit int) ([]FrameHit, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	url := GetInstance().Config.GetAIURL()
	if url == "" {
		return nil, errors.New("search by description needs the Stasharr Tagger (Settings → AI markers)")
	}
	prompt := text
	lower := strings.ToLower(text)
	if !strings.HasPrefix(lower, "a ") && !strings.HasPrefix(lower, "an ") && !strings.HasPrefix(lower, "the ") {
		prompt = "a photo of " + text
	}
	res, err := taggerPost(ctx, url, "/embed_text", map[string]any{"texts": []string{prompt}})
	if err != nil {
		return nil, err
	}
	if len(res.Vectors) == 0 {
		return nil, errors.New("the tagger sent no vector")
	}
	q := res.Vectors[0]
	idx, err := frameIndexCache.get(ctx, res.Model)
	if err != nil {
		return nil, err
	}
	if idx.dim == 0 || len(q) != idx.dim {
		return []FrameHit{}, nil
	}
	qf := make([]float32, len(q))
	for i, x := range q {
		qf[i] = float32(x)
	}

	best := map[int32]FrameHit{}
	n := len(idx.scenes)
	for f := 0; f < n; f++ {
		v := idx.vecs[f*idx.dim : (f+1)*idx.dim]
		var dot float32
		for i, x := range v {
			dot += float32(x) * qf[i]
		}
		score := float64(dot * idx.scales[f])
		s := idx.scenes[f]
		if h, ok := best[s]; !ok || score > h.Score {
			best[s] = FrameHit{SceneID: int(s), Seconds: float64(idx.secs[f]), Score: score}
		}
	}
	hits := make([]FrameHit, 0, len(best))
	for _, h := range best {
		hits = append(hits, h)
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}

// SearchModel returns the tagger's current model name, or "".
func SearchModel(ctx context.Context) string {
	url := GetInstance().Config.GetAIURL()
	if url == "" {
		return ""
	}
	st, err := GetTaggerStatus(ctx, url)
	if err != nil || !st.OK {
		return ""
	}
	return st.Model
}

// SaveFrame writes a 480 pixel wide picture of the video at sec.
func SaveFrame(ctx context.Context, src string, sec float64, out string) error {
	args := []string{"-hide_banner", "-nostats", "-v", "error", "-y",
		"-ss", fmt.Sprintf("%.2f", sec), "-i", src,
		"-frames:v", "1", "-vf", "scale=480:-2", "-q:v", "5",
		out}
	if _, err := runFFmpegCapture(ctx, args); err != nil {
		_ = os.Remove(out)
		return err
	}
	return nil
}
