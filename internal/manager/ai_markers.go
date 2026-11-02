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
}

// DefaultAILabels is the starting list. Admins can rename, reword, turn off
// or add labels.
var DefaultAILabels = []AILabel{
	{Name: "Kissing", Prompt: "a photo of two people kissing", Enabled: true, Marker: true},
	{Name: "Undressing", Prompt: "a photo of a person taking off their clothes", Enabled: true, Marker: true},
	{Name: "Blowjob", Prompt: "a photo of a woman performing oral sex on a man, a blowjob", Enabled: true, Marker: true},
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
	{Name: "Cumshot", Prompt: "a photo of a man ejaculating, a cumshot", Enabled: true, Marker: true},
	{Name: "Talking", Prompt: "a photo of people with clothes on, talking", Enabled: true, Marker: false},
	{Name: "Title card", Prompt: "a title screen with text or a logo", Enabled: true, Marker: false},
	{Name: "Scenery", Prompt: "a photo of a room or a place with nobody in it", Enabled: true, Marker: false},
}

// AILabels returns the saved labels, or the defaults.
func AILabels(cfg *config.Config) []AILabel {
	if s := cfg.GetAILabelsJSON(); s != "" {
		var labels []AILabel
		if err := json.Unmarshal([]byte(s), &labels); err == nil && len(labels) > 0 {
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
	Labels []string `json:"labels"`
	Images []string `json:"images"`
}

type classifyResponse struct {
	Scores [][]float64 `json:"scores"`
	Detail string      `json:"detail"`
}

func classify(ctx context.Context, url string, prompts []string, frames [][]byte) ([][]float64, error) {
	reqBody := classifyRequest{Labels: prompts}
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
	return out.Scores, nil
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
}

// planAIMarkers turns per-frame scores into marker segments. scores has one
// row per frame (frame i is at i*interval seconds) and one column per label.
func planAIMarkers(scores [][]float64, labels []AILabel, interval, minConfidence, minSeconds float64) []AISegment {
	n := len(scores)
	if n == 0 || len(labels) == 0 {
		return nil
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
		if best < 0 || !labels[best].Marker || bestScore < minConfidence {
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
			ret = append(ret, AISegment{Label: labels[l].Name, Start: round2(start), End: round2(end), Score: math.Round(sum/float64(cnt)*1000) / 1000})
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
}

func (j *AIMarkerJob) Execute(ctx context.Context, progress *job.Progress) error {
	mgr := GetInstance()
	cfg := mgr.Config
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
			ids, err = sqlite.AISceneIDsToDo(ctx)
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
			default:
				logger.Errorf("[ai markers] scene %d: %v", id, err)
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
	prompts := make([]string, len(labels))
	for i, l := range labels {
		prompts[i] = l.Prompt
	}

	started := time.Now()
	frames, err := sampleFrames(ctx, vf.Path, interval)
	if err != nil {
		return 0, err
	}
	var scores [][]float64
	const batch = 32
	for i := 0; i < len(frames); i += batch {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		end := min(i+batch, len(frames))
		s, err := classify(ctx, url, prompts, frames[i:end])
		if err != nil {
			return 0, err
		}
		scores = append(scores, s...)
	}

	segments := planAIMarkers(scores, labels, interval, cfg.GetAIMinConfidence(), cfg.GetAIMinSeconds())
	logger.Debugf("[ai markers] scene %d: %d frames in %s", sceneID, len(frames), time.Since(started).Round(time.Second))

	err = r.WithTxn(ctx, func(ctx context.Context) error {
		if cfg.GetAIReview() {
			list := make([]sqlite.AIMarkerSuggestion, len(segments))
			for i, s := range segments {
				list[i] = sqlite.AIMarkerSuggestion{Label: s.Label, Seconds: s.Start, EndSeconds: s.End, Score: s.Score}
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
