package manager

// Tag clips: every tag without its own picture gets a short silent loop cut
// from its best scene, shown on category tiles on the site, the Tags pages
// and the Android app.
//
// The scene is the best rated, most played one with the tag, preferring
// scenes with a marker for the tag (yours or an AI marker); the clip starts
// just after that marker. Without a marker it takes the scene's first reel
// moment, or the busiest part found the same way reels find theirs.

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sqlite"
)

const (
	tagClipSeconds = 4.0
	// the action usually starts a moment after the marker
	tagClipMarkerLead = 2.0
	tagClipFilter     = "scale=640:360:force_original_aspect_ratio=increase,crop=640:360,setsar=1"
)

// TagClipPaths returns where a tag's clip and still picture are kept.
func TagClipPaths(tagID int) (video, still string) {
	dir := GetInstance().Paths.Generated.TagClips
	base := filepath.Join(dir, strconv.Itoa(tagID))
	return base + ".mp4", base + ".jpg"
}

// RemoveTagClip deletes a tag's clip files.
func RemoveTagClip(tagID int) {
	v, s := TagClipPaths(tagID)
	_ = os.Remove(v)
	_ = os.Remove(s)
}

// TagClipJob makes clips for tags.
type TagClipJob struct {
	// TagIDs to make clips for; empty for every tag that needs one.
	TagIDs []int
	// Redo replaces existing clips.
	Redo bool
}

func (j *TagClipJob) Execute(ctx context.Context, progress *job.Progress) error {
	mgr := GetInstance()
	r := mgr.Repository

	ids := j.TagIDs
	redo := j.Redo || len(j.TagIDs) > 0
	if len(ids) == 0 {
		if err := r.WithReadTxn(ctx, func(ctx context.Context) error {
			var err error
			ids, err = sqlite.TagIDsNeedingClips(ctx, j.Redo)
			return err
		}); err != nil {
			return err
		}
		if j.Redo {
			// start afresh, so the old choices do not steer the new ones;
			// the old files stay until each tag's new clip replaces them
			if err := r.WithTxn(ctx, func(ctx context.Context) error {
				return sqlite.DeleteAllTagClips(ctx)
			}); err != nil {
				return err
			}
		}
	}
	if err := fsutil.EnsureDir(mgr.Paths.Generated.TagClips); err != nil {
		return fmt.Errorf("making the tag clips folder: %w", err)
	}

	// with the Stasharr Tagger running, each tag's moment is found by
	// looking for the tag in the scene's frames
	tagger := ""
	if url := mgr.Config.GetAIURL(); url != "" {
		if st, err := GetTaggerStatus(ctx, url); err == nil && st.OK {
			tagger = url
		} else {
			logger.Infof("[tag clips] the Stasharr Tagger is not ready; using reel moments instead")
		}
	}

	progress.SetTotal(len(ids))
	made := 0
	for _, id := range ids {
		if job.IsCancelled(ctx) {
			return nil
		}
		progress.ExecuteTask(fmt.Sprintf("Tag clip for tag %d", id), func() {
			ok, err := MakeTagClip(ctx, id, redo, tagger)
			switch {
			case err == nil && ok:
				made++
			case err == nil:
			case errors.Is(err, context.Canceled):
			default:
				logger.Warnf("[tag clips] tag %d: %v", id, err)
			}
		})
		progress.Increment()
	}
	logger.Infof("[tag clips] made %d clip(s)", made)
	return nil
}

// MakeTagClip cuts the clip for one tag. tagger is the Stasharr Tagger's
// address, or empty to do without it. It reports whether a clip was made.
func MakeTagClip(ctx context.Context, tagID int, redo bool, tagger string) (bool, error) {
	mgr := GetInstance()
	r := mgr.Repository
	video, still := TagClipPaths(tagID)

	if !redo {
		if _, err := os.Stat(video); err == nil {
			return false, nil
		}
	}

	var name string
	var candidates []sqlite.TagClipCandidate
	if err := r.WithReadTxn(ctx, func(ctx context.Context) error {
		t, err := r.Tag.Find(ctx, tagID)
		if err != nil {
			return err
		}
		if t == nil {
			return fmt.Errorf("tag %d not found", tagID)
		}
		name = t.Name
		candidates, err = sqlite.TagClipCandidates(ctx, tagID, 6)
		return err
	}); err != nil {
		return false, err
	}

	for _, c := range candidates {
		vf, reels, used, err := tagClipScene(ctx, c.SceneID, tagID)
		if err != nil {
			return false, err
		}
		if vf == nil || vf.IsAudioOnly() || vf.Duration < tagClipSeconds+2 {
			continue
		}
		start := tagClipStart(ctx, tagClipChoice{
			tagID: tagID, name: name, vf: vf, marker: c.Marker, reels: reels, used: used, tagger: tagger,
		})
		if err := cutTagClip(ctx, vf.Path, start, video, still); err != nil {
			logger.Debugf("[tag clips] tag %d, scene %d: %v", tagID, c.SceneID, err)
			continue
		}
		err = r.WithTxn(ctx, func(ctx context.Context) error {
			return sqlite.SaveTagClip(ctx, sqlite.TagClip{TagID: tagID, SceneID: c.SceneID, Seconds: start})
		})
		return err == nil, err
	}
	return false, nil
}

// tagClipScene loads the scene, its reels, and where other tags' clips from
// the same scene start.
func tagClipScene(ctx context.Context, sceneID, tagID int) (*models.VideoFile, []*models.Reel, []float64, error) {
	r := GetInstance().Repository
	var vf *models.VideoFile
	var reels []*models.Reel
	var used []float64
	err := r.WithReadTxn(ctx, func(ctx context.Context) error {
		s, err := r.Scene.Find(ctx, sceneID)
		if err != nil || s == nil {
			return err
		}
		if err := s.LoadPrimaryFile(ctx, r.File); err != nil {
			return err
		}
		vf = s.Files.Primary()
		if reels, err = r.Reel.FindBySceneID(ctx, sceneID); err != nil {
			return err
		}
		used, err = sqlite.TagClipTimesInScene(ctx, sceneID, tagID)
		return err
	})
	return vf, reels, used, err
}

type tagClipChoice struct {
	tagID  int
	name   string
	vf     *models.VideoFile
	marker *float64
	reels  []*models.Reel
	// where other tags' clips from this scene start
	used   []float64
	tagger string
}

// clips from the same scene start at least this far apart
const tagClipSpacing = 20.0

func (c tagClipChoice) free(t float64) bool {
	for _, u := range c.used {
		if math.Abs(u-t) < tagClipSpacing {
			return false
		}
	}
	return true
}

// tagClipStart picks where the clip begins, in this order:
//  1. just after a marker for the tag;
//  2. the frame that looks most like the tag, when the Stasharr Tagger runs;
//  3. a reel moment or busy part that no other tag's clip uses yet;
//  4. a spot picked from the tag's name, so tags still differ.
func tagClipStart(ctx context.Context, c tagClipChoice) float64 {
	limit := c.vf.Duration - tagClipSeconds - 0.5
	clamp := func(v float64) float64 { return round2(math.Max(0, math.Min(v, limit))) }

	if c.marker != nil {
		return clamp(*c.marker + tagClipMarkerLead)
	}

	if c.tagger != "" {
		if t, ok := tagMomentFromFrames(ctx, c); ok {
			return clamp(t)
		}
	}

	var moments []float64
	for _, rl := range c.reels {
		for _, seg := range rl.Segments {
			moments = append(moments, seg.Start+(seg.End-seg.Start-tagClipSeconds)/2)
		}
	}
	if scores, err := analysisScores(ctx, c.vf); err == nil && len(scores) > 0 {
		for _, p := range pickPeaks(scores, 6, tagClipSeconds, c.vf.Duration) {
			moments = append(moments, p.Start)
		}
	}
	// different tags start from different moments in the list
	if n := len(moments); n > 0 {
		off := int(uint32(c.tagID)*2654435761>>8) % n
		for i := 0; i < n; i++ {
			m := clamp(moments[(off+i)%n])
			if c.free(m) {
				return m
			}
		}
	}

	// spread over the middle 70% of the scene by the tag's id
	frac := 0.15 + 0.7*float64(uint32(c.tagID)*2654435761%1000)/1000
	for i := 0; i < 10; i++ {
		m := clamp(c.vf.Duration * frac)
		if c.free(m) {
			return m
		}
		frac = math.Mod(frac+0.137, 0.7) + 0.15
	}
	return clamp(c.vf.Duration * frac)
}

// at most this many frames are looked at to find a tag's moment
const tagClipMaxFrames = 150

// tagMomentFromFrames looks through the scene's frames with the Stasharr
// Tagger and returns the time that looks most like the tag.
func tagMomentFromFrames(ctx context.Context, c tagClipChoice) (float64, bool) {
	interval := math.Max(6, c.vf.Duration/tagClipMaxFrames)
	frames, err := sampleFrames(ctx, c.vf.Path, interval)
	if err != nil || len(frames) == 0 {
		return 0, false
	}

	// the tag's description: the AI label's wording when there is one,
	// otherwise its name; the other labels soak up talking, titles and empty
	// rooms
	prompts := []string{"a photo of " + strings.ToLower(strings.TrimSpace(c.name))}
	for _, l := range AILabels(GetInstance().Config) {
		switch {
		case strings.EqualFold(l.Name, c.name) && strings.TrimSpace(l.Prompt) != "":
			prompts[0] = l.Prompt
		case l.Enabled && !l.Marker && strings.TrimSpace(l.Prompt) != "":
			prompts = append(prompts, l.Prompt)
		}
	}
	if len(prompts) == 1 {
		prompts = append(prompts, "a photo of people with clothes on, talking", "a title screen with text or a logo")
	}

	var scores [][]float64
	for i := 0; i < len(frames); i += 32 {
		end := min(i+32, len(frames))
		s, err := classify(ctx, c.tagger, prompts, frames[i:end])
		if err != nil {
			logger.Debugf("[tag clips] tagger: %v", err)
			return 0, false
		}
		scores = append(scores, s...)
	}

	// smooth with the neighbours, then take the best free spot
	best, bestScore := -1.0, -1.0
	for i := range scores {
		sum, n := 0.0, 0.0
		for j := i - 1; j <= i+1; j++ {
			if j >= 0 && j < len(scores) && len(scores[j]) > 0 {
				sum += scores[j][0]
				n++
			}
		}
		t := float64(i)*interval - 1
		if n == 0 || !c.free(t) {
			continue
		}
		if v := sum / n; v > bestScore {
			best, bestScore = t, v
		}
	}
	if best < 0 {
		return 0, false
	}
	logger.Debugf("[tag clips] %q: best frame at %.0fs (%.2f)", c.name, best, bestScore)
	return best, true
}

// cutTagClip writes a 4 second 640×360 silent loop and a still picture.
func cutTagClip(ctx context.Context, src string, start float64, video, still string) error {
	tmp := video + ".part.mp4"
	defer os.Remove(tmp)
	at := strconv.FormatFloat(start, 'f', 2, 64)
	args := []string{"-hide_banner", "-nostats", "-v", "error", "-y",
		"-ss", at, "-i", src,
		"-t", strconv.FormatFloat(tagClipSeconds, 'f', 1, 64),
		"-an", "-sn", "-dn",
		"-vf", tagClipFilter + ",fps=24",
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "27",
		"-pix_fmt", "yuv420p", "-profile:v", "main",
		"-movflags", "+faststart",
		"-threads", "2",
		tmp}
	if _, err := runFFmpegCapture(ctx, args); err != nil {
		return err
	}
	if fi, err := os.Stat(tmp); err != nil || fi.Size() < 1024 {
		return errors.New("the clip came out empty")
	}
	if err := os.Rename(tmp, video); err != nil {
		return err
	}
	stillArgs := []string{"-hide_banner", "-nostats", "-v", "error", "-y",
		"-ss", strconv.FormatFloat(start+1, 'f', 2, 64), "-i", src,
		"-frames:v", "1", "-vf", tagClipFilter, "-q:v", "4",
		still}
	if _, err := runFFmpegCapture(ctx, stillArgs); err != nil {
		logger.Debugf("[tag clips] still picture: %v", err)
	}
	return nil
}
