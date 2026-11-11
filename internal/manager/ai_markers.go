package manager

// AI markers: Stasharr looks at frames from a scene with the Stasharr Tagger
// (a separate container running an OpenCLIP model on the CPU or an NVIDIA
// card) and turns what it sees into markers.
//
//  1. ffmpeg pulls one small frame every few seconds, decoding key frames
//     only, so it is quick and light.
//  2. The tagger scores each frame against a description of every label.
//  3. Scores are smoothed over neighbouring frames; a label that stays on
//     top, confidently, for long enough becomes a suggested marker.
//  4. Suggestions wait for an admin to approve them, or become markers
//     straight away when review is off.

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
	"strconv"
	"strings"
	"time"

	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sqlite"
)

// AILabel is one thing the tagger looks for.
type AILabel struct {
	// Name is the marker's title and tag.
	Name string `json:"name"`
	// Prompt describes it to the model in plain English.
	Prompt string `json:"prompt"`
	// Enabled labels are looked for.
	Enabled bool `json:"enabled"`
	// Marker labels become markers; the others only soak up frames that
	// are not of interest (talking, title cards…) so they are not forced
	// into a marker label.
	Marker bool `json:"marker"`
	// When limits where in a scene the label can be found: "end" for the
	// last 30% (a cumshot), "start" for the first 30%, "" for anywhere.
	When string `json:"when,omitempty"`
	// MinConfidence overrides the general "how sure" for this label; 0
	// uses the general one.
	MinConfidence float64 `json:"min_confidence,omitempty"`
}

// AILabelPartOfScene is how much of a scene counts as its start or end.
const AILabelPartOfScene = 0.3

// DefaultAILabels is the starting list. Admins can rename, reword, turn off
// or add labels.
var DefaultAILabels = []AILabel{
	{Name: "Kissing", Prompt: "a photo of two people kissing", Enabled: true, Marker: true},
	{Name: "Undressing", Prompt: "a photo of a person taking off their clothes", Enabled: true, Marker: true},
	{Name: "Blowjob", Prompt: "a photo of a woman performing oral sex on a man, a blowjob | a close-up photo of a woman sucking a penis", Enabled: true, Marker: true},
	{Name: "Cunnilingus", Prompt: "a photo of a person performing oral sex on a woman", Enabled: true, Marker: true},
	{Name: "Handjob", Prompt: "a photo of a woman giving a handjob", Enabled: true, Marker: true},
	{Name: "Titjob", Prompt: "a photo of a titjob, a penis between breasts", Enabled: true, Marker: true},
	{Name: "Missionary", Prompt: "a photo of sex in the missionary position, woman lying on her back", Enabled: true, Marker: true},
	{Name: "Cowgirl", Prompt: "a photo of a woman riding on top of a man, cowgirl position, facing him", Enabled: true, Marker: true},
	{Name: "Reverse Cowgirl", Prompt: "a photo of a woman riding on top of a man facing away from him, reverse cowgirl position", Enabled: true, Marker: true},
	{Name: "Doggy Style", Prompt: "a photo of sex from behind, doggy style, woman on her hands and knees", Enabled: true, Marker: true},
	{Name: "Spooning", Prompt: "a photo of two people having sex lying on their sides, spooning", Enabled: true, Marker: true},
	{Name: "Standing Sex", Prompt: "a photo of two people having sex while standing", Enabled: true, Marker: true},
	{Name: "Masturbation", Prompt: "a photo of a woman masturbating alone", Enabled: true, Marker: true},
	{Name: "Cumshot", Prompt: "a close-up photo of a woman's face covered in white semen | a photo of a man ejaculating on a woman | a photo of white cum on a woman's breasts or belly | a woman with her mouth open and cum on her tongue", Enabled: true, Marker: true, When: "end", MinConfidence: 0.5},
	{Name: "Talking", Prompt: "a photo of people with clothes on, talking", Enabled: true, Marker: false},
	{Name: "Title card", Prompt: "a title screen with text or a logo", Enabled: true, Marker: false},
	{Name: "Scenery", Prompt: "a photo of a room or a place with nobody in it", Enabled: true, Marker: false},
	// close-ups are what the model mistook for a cumshot most; giving them
	// a label of their own soaks them up
	{Name: "Face close-up", Prompt: "a portrait photo of a woman's face, smiling at the camera | a close-up photo of a woman's face and hair", Enabled: true, Marker: false},
	{Name: "Body close-up", Prompt: "an extreme close-up photo of skin | a blurry close-up photo of a body", Enabled: true, Marker: false},
}

// aiLabelsVersion is the version of DefaultAILabels. Saved lists from an
// older version get the new wording for labels whose wording was never
// changed, and the new helper labels.
const AILabelsVersion = 2

// aiOldPrompts are the version 1 wordings that version 2 replaced.
var aiOldPrompts = map[string]string{
	"blowjob": "a photo of a woman performing oral sex on a man, a blowjob",
	"cumshot": "a photo of a man ejaculating, a cumshot",
}

func upgradeAILabels(labels []AILabel) []AILabel {
	have := map[string]bool{}
	for i, l := range labels {
		key := strings.ToLower(l.Name)
		have[key] = true
		if old, ok := aiOldPrompts[key]; ok && strings.TrimSpace(l.Prompt) == old {
			for _, d := range DefaultAILabels {
				if strings.EqualFold(d.Name, l.Name) {
					labels[i].Prompt = d.Prompt
					labels[i].When = d.When
					labels[i].MinConfidence = d.MinConfidence
				}
			}
		}
	}
	for _, d := range DefaultAILabels {
		if !d.Marker && !have[strings.ToLower(d.Name)] {
			labels = append(labels, d)
		}
	}
	return labels
}

// AILabels returns the saved labels, or the defaults.
func AILabels(cfg *config.Config) []AILabel {
	if s := cfg.GetAILabelsJSON(); s != "" {
		var labels []AILabel
		if err := json.Unmarshal([]byte(s), &labels); err == nil && len(labels) > 0 {
			if cfg.GetAILabelsVersion() < AILabelsVersion {
				labels = upgradeAILabels(labels)
			}
			return labels
		}
	}
	return append([]AILabel{}, DefaultAILabels...)
}

// ---------------------------------------------------------------- the tagger

// TaggerStatus is what the tagger reports about itself.
type TaggerStatus struct {
	Reachable bool
	OK        bool    `json:"ok"`
	Loading   bool    `json:"loading"`
	Error     *string `json:"error"`
	Device    string  `json:"device"`
	GPU       *string `json:"gpu"`
	Model     string  `json:"model"`
	// Features lists what this tagger version can do.
	Features []string `json:"features"`
}

// Has reports whether the tagger can do something.
func (t *TaggerStatus) Has(feature string) bool {
	for _, f := range t.Features {
		if f == feature {
			return true
		}
	}
	return false
}

var taggerClient = &http.Client{Timeout: 10 * time.Minute}

// GetTaggerStatus asks the tagger whether it is ready.
func GetTaggerStatus(ctx context.Context, url string) (*TaggerStatus, error) {
	if url == "" {
		return nil, errors.New("no tagger address set")
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"/health", nil)
	if err != nil {
		return nil, err
	}
	res, err := taggerClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach the tagger at %s: %w", url, err)
	}
	defer res.Body.Close()
	st := &TaggerStatus{}
	if err := json.NewDecoder(res.Body).Decode(st); err != nil {
		return nil, fmt.Errorf("the tagger sent an unexpected answer: %w", err)
	}
	st.Reachable = true
	return st, nil
}

type classifyRequest struct {
	Labels  []string `json:"labels"`
	Images  []string `json:"images"`
	Vectors bool     `json:"vectors,omitempty"`
}

type classifyResponse struct {
	Scores  [][]float64 `json:"scores"`
	Vectors [][]float32 `json:"vectors"`
	Detail  string      `json:"detail"`
}

func classify(ctx context.Context, url string, prompts []string, frames [][]byte, vectors bool) (*classifyResponse, error) {
	reqBody := classifyRequest{Labels: prompts, Vectors: vectors}
	for _, f := range frames {
		reqBody.Images = append(reqBody.Images, base64.StdEncoding.EncodeToString(f))
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url+"/classify", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := taggerClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tagger: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 32<<20))
	var out classifyResponse
	_ = json.Unmarshal(body, &out)
	if res.StatusCode != http.StatusOK {
		msg := out.Detail
		if msg == "" {
			msg = strings.TrimSpace(string(body))
		}
		return nil, fmt.Errorf("tagger: %s: %s", res.Status, msg)
	}
	if len(out.Scores) != len(frames) {
		return nil, fmt.Errorf("tagger: expected %d answers, got %d", len(frames), len(out.Scores))
	}
	if len(out.Vectors) != len(frames) {
		out.Vectors = nil
	}
	return &out, nil
}

// ---------------------------------------------------------------- frames

// at most this many frames per scene; long films are looked at more sparsely
const maxAIFrames = 1500

// sampleFrames returns small JPEG frames every interval seconds.
func sampleFrames(ctx context.Context, path string, interval float64) ([][]byte, error) {
	args := []string{"-hide_banner", "-nostats", "-v", "error",
		// key frames only: far less decoding, timing within a few seconds
		"-skip_frame", "nokey",
		"-i", path,
		"-an", "-sn", "-dn",
		"-vf", fmt.Sprintf("fps=1/%s,scale=-2:256", strconv.FormatFloat(interval, 'f', 3, 64)),
		"-fps_mode", "vfr",
		"-threads", "2",
		"-f", "image2pipe", "-c:v", "mjpeg", "-q:v", "5",
		"-"}
	out, err := runFFmpegCapture(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("reading frames: %w", err)
	}
	return splitJPEGs(out), nil
}

// splitJPEGs cuts a stream of JPEG images apart. Inside a JPEG every 0xFF
// in the picture data is followed by 0x00, so the end marker FFD9 only
// appears at the end of each image.
func splitJPEGs(b []byte) [][]byte {
	var ret [][]byte
	for len(b) > 4 {
		start := bytes.Index(b, []byte{0xFF, 0xD8})
		if start < 0 {
			break
		}
		end := bytes.Index(b[start+2:], []byte{0xFF, 0xD9})
		if end < 0 {
			break
		}
		end += start + 4
		ret = append(ret, b[start:end])
		b = b[end:]
	}
	return ret
}

// ---------------------------------------------------------------- planning

// AISegment is a stretch of a scene that gets one marker.
type AISegment struct {
	Label string
	Start float64
	End   float64
	Score float64
	// first and last frame
	First, Last int
}

// planAIMarkers turns per-frame scores into marker segments. scores has one
// row per frame (frame i is at i*interval seconds) and one column per label.
func planAIMarkers(scores [][]float64, labels []AILabel, interval, minConfidence, minSeconds float64) []AISegment {
	n := len(scores)
	if n == 0 || len(labels) == 0 {
		return nil
	}
	// labels limited to part of the scene score nothing elsewhere, so
	// another label can win those frames
	length := float64(n) * interval
	for i := range scores {
		at := float64(i) * interval
		for l, lab := range labels {
			if l >= len(scores[i]) {
				continue
			}
			switch lab.When {
			case "end":
				if at < length*(1-AILabelPartOfScene) {
					scores[i][l] = 0
				}
			case "start":
				if at > length*AILabelPartOfScene {
					scores[i][l] = 0
				}
			}
		}
	}
	// smooth each frame with its neighbours so one odd frame does not
	// split a stretch
	top := make([]int, n)
	conf := make([]float64, n)
	for i := 0; i < n; i++ {
		best, bestScore := -1, 0.0
		for l := range labels {
			sum, cnt := 0.0, 0.0
			for j := i - 1; j <= i+1; j++ {
				if j < 0 || j >= n || l >= len(scores[j]) {
					continue
				}
				w := 1.0
				if j == i {
					w = 2
				}
				sum += scores[j][l] * w
				cnt += w
			}
			if cnt > 0 && sum/cnt > bestScore {
				best, bestScore = l, sum/cnt
			}
		}
		top[i], conf[i] = best, bestScore
		need := minConfidence
		if best >= 0 && labels[best].MinConfidence > 0 {
			need = labels[best].MinConfidence
		}
		if best < 0 || !labels[best].Marker || bestScore < need {
			top[i] = -1
		}
	}

	var ret []AISegment
	i := 0
	for i < n {
		if top[i] < 0 {
			i++
			continue
		}
		l := top[i]
		j, last := i, i
		sum, cnt := 0.0, 0
		gap := 0
		for j < n {
			switch {
			case top[j] == l:
				last = j
				sum += conf[j]
				cnt++
				gap = 0
			case gap < 1:
				// allow one stray frame in the middle of a stretch
				gap++
			default:
				j = n
				continue
			}
			j++
		}
		start := float64(i) * interval
		end := float64(last+1) * interval
		if end-start >= minSeconds {
			ret = append(ret, AISegment{Label: labels[l].Name, Start: round2(start), End: round2(end), Score: math.Round(sum/float64(cnt)*1000) / 1000, First: i, Last: last})
		}
		i = last + 1
	}
	return ret
}

// ---------------------------------------------------------------- the job

// AIMarkerJob looks at scenes and suggests or makes markers.
type AIMarkerJob struct {
	// SceneIDs to look at; empty for every scene not looked at yet.
	SceneIDs []int
	// Redo looks at every scene again (when SceneIDs is empty). Waiting
	// suggestions are replaced; markers already kept stay.
	Redo bool
}

func (j *AIMarkerJob) Execute(ctx context.Context, progress *job.Progress) error {
	mgr := GetInstance()
	cfg := mgr.Config
	if !cfg.GetAIEnabled() {
		return errors.New("AI markers are switched off (Settings → AI markers)")
	}
	url := cfg.GetAIURL()
	if url == "" {
		return errors.New("set the Stasharr Tagger address first (Settings → AI markers)")
	}
	st, err := GetTaggerStatus(ctx, url)
	if err != nil {
		return err
	}
	if !st.OK {
		if st.Error != nil {
			return fmt.Errorf("the tagger could not load its model: %s", *st.Error)
		}
		return errors.New("the tagger is still loading its model; try again in a few minutes")
	}

	ids := j.SceneIDs
	if len(ids) == 0 {
		if err := mgr.Repository.WithReadTxn(ctx, func(ctx context.Context) error {
			var err error
			if j.Redo {
				ids, err = sqlite.AllSceneIDs(ctx)
			} else {
				ids, err = sqlite.AISceneIDsToDo(ctx)
			}
			return err
		}); err != nil {
			return err
		}
	}

	progress.SetTotal(len(ids))
	for _, id := range ids {
		if job.IsCancelled(ctx) {
			logger.Info("[ai markers] stopped")
			return nil
		}
		progress.ExecuteTask(fmt.Sprintf("Looking at scene %d", id), func() {
			n, err := MarkSceneWithAI(ctx, id)
			switch {
			case err == nil:
				logger.Infof("[ai markers] scene %d: %d marker(s) found", id, n)
			case errors.Is(err, context.Canceled):
				return
			default:
				logger.Errorf("[ai markers] scene %d: %v", id, err)
				return
			}
			// while the tagger is looking at this scene anyway, let it pick
			// the scene's reels too
			if cfg.GetReelsEnabled() && cfg.GetReelsAI() {
				progress.ExecuteTask(fmt.Sprintf("Reels for scene %d", id), func() {
					if _, err := GenerateReels(ctx, id, cfg.GetReelDuration(), cfg.GetReelCount(), true); err != nil &&
						!errors.Is(err, errReelAudio) && !errors.Is(err, context.Canceled) {
						logger.Warnf("[reels] scene %d: %v", id, err)
					}
				})
			}
		})
		progress.Increment()
	}
	return nil
}

// MarkSceneWithAI looks at one scene and returns how many markers it found.
func MarkSceneWithAI(ctx context.Context, sceneID int) (int, error) {
	mgr := GetInstance()
	cfg := mgr.Config
	r := mgr.Repository
	url := cfg.GetAIURL()

	var vf *models.VideoFile
	if err := r.WithReadTxn(ctx, func(ctx context.Context) error {
		s, err := r.Scene.Find(ctx, sceneID)
		if err != nil {
			return err
		}
		if s == nil {
			return fmt.Errorf("scene %d not found", sceneID)
		}
		if err := s.LoadPrimaryFile(ctx, r.File); err != nil {
			return err
		}
		vf = s.Files.Primary()
		return nil
	}); err != nil {
		return 0, err
	}
	if vf == nil || vf.IsAudioOnly() || vf.Duration <= 0 {
		// nothing to look at; remember it so it is not tried again
		return 0, r.WithTxn(ctx, func(ctx context.Context) error {
			return sqlite.AIReplaceSuggestions(ctx, sceneID, nil, 0)
		})
	}

	interval := cfg.GetAIInterval()
	if vf.Duration/interval > maxAIFrames {
		interval = vf.Duration / maxAIFrames
	}

	labels := []AILabel{}
	for _, l := range AILabels(cfg) {
		if l.Enabled && strings.TrimSpace(l.Prompt) != "" && strings.TrimSpace(l.Name) != "" {
			labels = append(labels, l)
		}
	}
	if len(labels) == 0 {
		return 0, errors.New("no labels are turned on")
	}
	st, err := GetTaggerStatus(ctx, url)
	if err != nil {
		return 0, err
	}
	ensemble, wantVectors := st.Has("ensemble"), st.Has("vectors")
	prompts := make([]string, len(labels))
	for i, l := range labels {
		prompts[i] = l.Prompt
		if !ensemble {
			// an older tagger reads "a | b" as one sentence; use the first
			prompts[i] = strings.TrimSpace(strings.Split(l.Prompt, "|")[0])
		}
	}

	started := time.Now()
	frames, err := sampleFrames(ctx, vf.Path, interval)
	if err != nil {
		return 0, err
	}
	var scores [][]float64
	var vectors [][]float32
	const batch = 32
	for i := 0; i < len(frames); i += batch {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		end := min(i+batch, len(frames))
		res, err := classify(ctx, url, prompts, frames[i:end], wantVectors)
		if err != nil {
			return 0, err
		}
		scores = append(scores, res.Scores...)
		if res.Vectors == nil {
			wantVectors = false
		}
		vectors = append(vectors, res.Vectors...)
	}
	if !wantVectors || len(vectors) != len(scores) {
		vectors = nil
	}

	// what the admin kept and threw away before
	if vectors != nil {
		var examples []sqlite.AIExample
		if err := r.WithReadTxn(ctx, func(ctx context.Context) error {
			var err error
			examples, err = sqlite.AIExamples(ctx, st.Model, aiExamplesPerLabel)
			return err
		}); err != nil {
			logger.Warnf("[ai markers] reading examples: %v", err)
		}
		learnFromExamples(scores, vectors, labels, examples)
	}

	segments := planAIMarkers(scores, labels, interval, cfg.GetAIMinConfidence(), cfg.GetAIMinSeconds())
	logger.Debugf("[ai markers] scene %d: %d frames in %s", sceneID, len(frames), time.Since(started).Round(time.Second))

	err = r.WithTxn(ctx, func(ctx context.Context) error {
		if cfg.GetAIReview() {
			list := make([]sqlite.AIMarkerSuggestion, len(segments))
			for i, s := range segments {
				list[i] = sqlite.AIMarkerSuggestion{Label: s.Label, Seconds: s.Start, EndSeconds: s.End, Score: s.Score}
				if vectors != nil {
					// remember what it looked like, to learn from the review
					q, scale := quantize(meanVector(vectors[s.First : s.Last+1]))
					model := st.Model
					list[i].Model, list[i].Scale, list[i].Vec = &model, &scale, q
				}
			}
			return sqlite.AIReplaceSuggestions(ctx, sceneID, list, len(segments))
		}
		for _, s := range segments {
			if _, err := SaveAIMarker(ctx, r, sceneID, s.Label, s.Start, s.End); err != nil {
				return err
			}
		}
		return sqlite.AIReplaceSuggestions(ctx, sceneID, nil, len(segments))
	})
	return len(segments), err
}

// SaveAIMarker makes a marker titled and tagged with the label, creating
// the tag if needed. A marker with the same tag close by is not repeated.
// It must run inside a write transaction. It reports whether one was made.
func SaveAIMarker(ctx context.Context, r models.Repository, sceneID int, label string, start, end float64) (bool, error) {
	tag, err := r.Tag.FindByName(ctx, label, true)
	if err != nil {
		return false, err
	}
	if tag == nil {
		t := models.NewTag()
		t.Name = label
		if err := r.Tag.Create(ctx, &models.CreateTagInput{Tag: &t}); err != nil {
			return false, fmt.Errorf("creating tag %q: %w", label, err)
		}
		tag = &t
	}
	existing, err := r.SceneMarker.FindBySceneID(ctx, sceneID)
	if err != nil {
		return false, err
	}
	for _, m := range existing {
		if m.PrimaryTagID == tag.ID && math.Abs(m.Seconds-start) < 20 {
			return false, nil
		}
	}
	m := models.NewSceneMarker()
	m.Title = label
	m.Seconds = start
	if end > start {
		e := end
		m.EndSeconds = &e
	}
	m.PrimaryTagID = tag.ID
	m.SceneID = sceneID
	if err := r.SceneMarker.Create(ctx, &m); err != nil {
		return false, err
	}
	return true, nil
}

// ---------------------------------------------------------------- learning

// how many kept and thrown away examples per label are used
const aiExamplesPerLabel = 300

// a frame must look at least this much like an example for it to count
const aiExampleMinSimilarity = 0.78

type aiExampleSet struct {
	kept, thrown [][]float32
}

func dequantize(vec []byte, scale float64) []float32 {
	out := make([]float32, len(vec))
	var norm float64
	for i, b := range vec {
		x := float64(int8(b)) * scale
		out[i] = float32(x)
		norm += x * x
	}
	if norm > 0 {
		n := float32(math.Sqrt(norm))
		for i := range out {
			out[i] /= n
		}
	}
	return out
}

func meanVector(list [][]float32) []float64 {
	if len(list) == 0 {
		return nil
	}
	out := make([]float64, len(list[0]))
	for _, v := range list {
		for i := range out {
			if i < len(v) {
				out[i] += float64(v[i])
			}
		}
	}
	var norm float64
	for _, x := range out {
		norm += x * x
	}
	if norm > 0 {
		n := math.Sqrt(norm)
		for i := range out {
			out[i] /= n
		}
	}
	return out
}

func nearest(v []float32, list [][]float32) float64 {
	best := -1.0
	for _, e := range list {
		if len(e) != len(v) {
			continue
		}
		var dot float32
		for i, x := range e {
			dot += x * v[i]
		}
		if float64(dot) > best {
			best = float64(dot)
		}
	}
	return best
}

// learnFromExamples changes the scores of frames that look like markers
// the admin kept (up) or threw away (down) for the same label. One thrown
// away "cumshot" that was really a close-up of a face then stops similar
// close-ups from becoming cumshots.
func learnFromExamples(scores [][]float64, vectors [][]float32, labels []AILabel, examples []sqlite.AIExample) {
	if len(examples) == 0 {
		return
	}
	sets := map[string]*aiExampleSet{}
	for _, e := range examples {
		key := strings.ToLower(e.Label)
		set := sets[key]
		if set == nil {
			set = &aiExampleSet{}
			sets[key] = set
		}
		v := dequantize(e.Vec, e.Scale)
		if e.Kept {
			set.kept = append(set.kept, v)
		} else {
			set.thrown = append(set.thrown, v)
		}
	}
	for l, lab := range labels {
		set := sets[strings.ToLower(lab.Name)]
		if set == nil || !lab.Marker {
			continue
		}
		for f := range scores {
			if l >= len(scores[f]) || f >= len(vectors) {
				continue
			}
			k := nearest(vectors[f], set.kept)
			t := nearest(vectors[f], set.thrown)
			switch {
			case t >= aiExampleMinSimilarity && t > k+0.01:
				scores[f][l] *= 0.15
			case k >= aiExampleMinSimilarity && k > t+0.01:
				scores[f][l] = math.Min(1, scores[f][l]*1.4+0.1)
			}
		}
	}
}

// AIExampleFromSuggestion turns a reviewed suggestion into an example. A
// suggestion made before the tagger sent vectors uses the scene's search
// vectors for the same stretch, when the scene is indexed.
func AIExampleFromSuggestion(ctx context.Context, s *sqlite.AIMarkerSuggestion, kept bool) (*sqlite.AIExample, error) {
	if len(s.Vec) > 0 && s.Model != nil && s.Scale != nil {
		return &sqlite.AIExample{Label: s.Label, Kept: kept, Model: *s.Model, Scale: *s.Scale, Vec: s.Vec}, nil
	}
	model, list, err := sqlite.FrameVectorsBetween(ctx, s.SceneID, s.Seconds, s.EndSeconds)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	vecs := make([][]float32, len(list))
	for i, v := range list {
		vecs[i] = dequantize(v.Vec, v.Scale)
	}
	q, scale := quantize(meanVector(vecs))
	return &sqlite.AIExample{Label: s.Label, Kept: kept, Model: model, Scale: scale, Vec: q}, nil
}
