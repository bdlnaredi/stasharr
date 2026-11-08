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
	}
	if err := fsutil.EnsureDir(mgr.Paths.Generated.TagClips); err != nil {
		return fmt.Errorf("making the tag clips folder: %w", err)
	}

	progress.SetTotal(len(ids))
	made := 0
	for _, id := range ids {
		if job.IsCancelled(ctx) {
			return nil
		}
		progress.ExecuteTask(fmt.Sprintf("Tag clip for tag %d", id), func() {
			ok, err := MakeTagClip(ctx, id, redo)
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

// MakeTagClip cuts the clip for one tag. It reports whether one was made.
func MakeTagClip(ctx context.Context, tagID int, redo bool) (bool, error) {
	mgr := GetInstance()
	r := mgr.Repository
	video, still := TagClipPaths(tagID)

	if !redo {
		if _, err := os.Stat(video); err == nil {
			return false, nil
		}
	}

	var candidates []sqlite.TagClipCandidate
	if err := r.WithReadTxn(ctx, func(ctx context.Context) error {
		var err error
		candidates, err = sqlite.TagClipCandidates(ctx, tagID, 6)
		return err
	}); err != nil {
		return false, err
	}

	for _, c := range candidates {
		vf, reels, err := tagClipScene(ctx, c.SceneID)
		if err != nil {
			return false, err
		}
		if vf == nil || vf.IsAudioOnly() || vf.Duration < tagClipSeconds+2 {
			continue
		}
		start := tagClipStart(ctx, vf, c.Marker, reels)
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

func tagClipScene(ctx context.Context, sceneID int) (*models.VideoFile, []*models.Reel, error) {
	r := GetInstance().Repository
	var vf *models.VideoFile
	var reels []*models.Reel
	err := r.WithReadTxn(ctx, func(ctx context.Context) error {
		s, err := r.Scene.Find(ctx, sceneID)
		if err != nil || s == nil {
			return err
		}
		if err := s.LoadPrimaryFile(ctx, r.File); err != nil {
			return err
		}
		vf = s.Files.Primary()
		reels, err = r.Reel.FindBySceneID(ctx, sceneID)
		return err
	})
	return vf, reels, err
}

// tagClipStart picks where the clip begins.
func tagClipStart(ctx context.Context, vf *models.VideoFile, marker *float64, reels []*models.Reel) float64 {
	limit := vf.Duration - tagClipSeconds - 0.5
	clamp := func(v float64) float64 { return math.Max(0, math.Min(v, limit)) }

	if marker != nil {
		return round2(clamp(*marker + tagClipMarkerLead))
	}
	for _, rl := range reels {
		if len(rl.Segments) > 0 {
			seg := rl.Segments[0]
			// the middle of the reel's moment
			return round2(clamp(seg.Start + (seg.End-seg.Start-tagClipSeconds)/2))
		}
	}
	if scores, err := analysisScores(ctx, vf); err == nil && len(scores) > 0 {
		if peaks := pickPeaks(scores, 1, tagClipSeconds, vf.Duration); len(peaks) > 0 {
			return round2(clamp(peaks[0].Start))
		}
	}
	return round2(clamp(vf.Duration * 0.35))
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
