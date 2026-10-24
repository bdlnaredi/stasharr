package manager

// Automatic short vertical reels. Each reel is one continuous clip of a
// scene (15 seconds by default), and a scene gets several reels (3 by
// default), each from a different high point.
//
// High points are found, cheaply, from:
//   - HEATMAP:  the intensity of an interactive funscript next to the video
//   - ANALYSIS: loudness of the audio track combined with on-screen motion,
//     measured by ffmpeg from the audio and the key frames only
//   - MARKERS:  the scene's markers (from you, or from ThePornDB / StashDB)
//     are a strong hint: the clip after a marker is preferred
//
// Each clip is letterboxed onto a blurred 9:16 background.

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/stashapp/stash/pkg/file/video"
	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
)

const (
	ReelSourceMarkers  = "MARKERS"
	ReelSourceHeatmap  = "HEATMAP"
	ReelSourceAnalysis = "ANALYSIS"
	ReelSourceEven     = "EVEN"

	reelWidth  = 720
	reelHeight = 1280
)

// ReelJob generates reels for a set of scenes.
type ReelJob struct {
	SceneIDs []int
	Duration float64
	// Count is the number of reels per scene. 0 uses the setting.
	Count     int
	Overwrite bool
}

func (j *ReelJob) Execute(ctx context.Context, progress *job.Progress) error {
	mgr := GetInstance()
	r := mgr.Repository

	duration := j.Duration
	if duration <= 0 {
		duration = mgr.Config.GetReelDuration()
	}
	count := j.Count
	if count <= 0 {
		count = mgr.Config.GetReelCount()
	}

	ids := j.SceneIDs
	if len(ids) == 0 {
		// every scene without a reel
		if err := r.WithReadTxn(ctx, func(ctx context.Context) error {
			existing, err := r.Reel.All(ctx)
			if err != nil {
				return err
			}
			has := map[int]bool{}
			for _, reel := range existing {
				has[reel.SceneID] = true
			}
			all, err := r.Scene.All(ctx)
			if err != nil {
				return err
			}
			for _, s := range all {
				if !has[s.ID] || j.Overwrite {
					ids = append(ids, s.ID)
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}

	progress.SetTotal(len(ids))
	for _, id := range ids {
		if job.IsCancelled(ctx) {
			logger.Info("Stopping reel generation due to user request")
			return nil
		}
		progress.ExecuteTask(fmt.Sprintf("Making reels for scene %d", id), func() {
			if _, err := GenerateReels(ctx, id, duration, count, j.Overwrite); err != nil {
				if errors.Is(err, errReelExists) || errors.Is(err, errReelAudio) {
					return
				}
				logger.Errorf("[reels] scene %d: %v", id, err)
			}
		})
		progress.Increment()
	}
	return nil
}

var (
	errReelExists = errors.New("scene already has reels")
	errReelAudio  = errors.New("audio files have no picture for a reel")
)

type reelPlan struct {
	source string
	// one clip per reel
	clips []models.ReelSegment
}

// GenerateReels makes up to count reels for one scene and stores them,
// replacing the scene's old reels.
func GenerateReels(ctx context.Context, sceneID int, duration float64, count int, overwrite bool) ([]*models.Reel, error) {
	mgr := GetInstance()
	r := mgr.Repository

	var (
		scene   *models.Scene
		vf      *models.VideoFile
		markers []*models.SceneMarker
		old     []*models.Reel
	)
	if err := r.WithReadTxn(ctx, func(ctx context.Context) error {
		var err error
		scene, err = r.Scene.Find(ctx, sceneID)
		if err != nil {
			return err
		}
		if scene == nil {
			return fmt.Errorf("scene %d not found", sceneID)
		}
		if err := scene.LoadPrimaryFile(ctx, r.File); err != nil {
			return err
		}
		vf = scene.Files.Primary()
		if vf == nil {
			return errors.New("scene has no video file")
		}
		markers, err = r.SceneMarker.FindBySceneID(ctx, sceneID)
		if err != nil {
			return err
		}
		old, err = r.Reel.FindBySceneID(ctx, sceneID)
		return err
	}); err != nil {
		return nil, err
	}

	if vf.IsAudioOnly() {
		return nil, errReelAudio
	}

	if len(old) > 0 && !overwrite {
		return nil, errReelExists
	}

	if vf.Duration <= 0 {
		return nil, errors.New("video duration unknown, scan the file first")
	}

	plan := planReels(ctx, vf, markers, duration, count)
	if len(plan.clips) == 0 {
		return nil, errors.New("no moments to use")
	}

	outDir := mgr.Paths.Generated.Reels
	if err := fsutil.EnsureDir(outDir); err != nil {
		return nil, err
	}

	stamp := time.Now().Unix()
	var made []*models.Reel
	cleanup := func() {
		for _, m := range made {
			_ = os.Remove(m.Path)
		}
	}
	for i, clip := range plan.clips {
		if job.IsCancelled(ctx) {
			cleanup()
			return nil, ctx.Err()
		}
		out := filepath.Join(outDir, fmt.Sprintf("%d_%d_%d.mp4", sceneID, stamp, i+1))
		if err := renderReel(ctx, vf, clip, out); err != nil {
			_ = os.Remove(out)
			cleanup()
			return nil, err
		}
		made = append(made, &models.Reel{
			SceneID:  sceneID,
			Path:     out,
			Duration: round2(clip.End - clip.Start),
			Width:    reelWidth,
			Height:   reelHeight,
			Source:   plan.source,
			Segments: []models.ReelSegment{clip},
		})
	}

	if err := r.WithTxn(ctx, func(ctx context.Context) error {
		for _, o := range old {
			if err := r.Reel.Destroy(ctx, o.ID); err != nil {
				return err
			}
		}
		// newest first in the feed: create the best-placed clip last
		for i := len(made) - 1; i >= 0; i-- {
			if err := r.Reel.Create(ctx, made[i]); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		cleanup()
		return nil, err
	}

	for _, o := range old {
		_ = os.Remove(o.Path)
	}

	logger.Infof("[reels] made %d reel(s) of %.0fs for %q from %s", len(made), duration, scene.GetTitle(), strings.ToLower(plan.source))
	return made, nil
}

// distinctMarkerTimes returns marker times at least minGap seconds apart.
func distinctMarkerTimes(markers []*models.SceneMarker, total, minGap float64) []float64 {
	var times []float64
	for _, m := range markers {
		if m.Seconds >= 0 && m.Seconds < total-1 {
			times = append(times, m.Seconds)
		}
	}
	sort.Float64s(times)
	var ret []float64
	for _, t := range times {
		if len(ret) == 0 || t-ret[len(ret)-1] >= minGap {
			ret = append(ret, t)
		}
	}
	return ret
}

// planReels picks count continuous clips of duration seconds from the most
// interesting, well separated parts of the video.
func planReels(ctx context.Context, vf *models.VideoFile, markers []*models.SceneMarker, duration float64, count int) *reelPlan {
	total := vf.Duration

	// short videos: one reel of the whole video
	if total <= duration+1 {
		return &reelPlan{source: ReelSourceEven, clips: []models.ReelSegment{{Start: 0, End: round2(total)}}}
	}

	// do not cram more clips in than fit with some room between them
	fit := int((total * 0.9) / (duration * 1.5))
	if fit < 1 {
		fit = 1
	}
	if count > fit {
		count = fit
	}
	if count < 1 {
		count = 1
	}

	var scores []float64
	source := ""

	if vf.Interactive {
		if s, err := funscriptScores(video.GetFunscriptPath(vf.Path), total); err == nil && len(s) > 0 {
			scores = zscores(s, true)
			source = ReelSourceHeatmap
		}
	}
	if scores == nil {
		s, err := analysisScores(ctx, vf)
		if err != nil {
			logger.Warnf("[reels] analysis of %s failed: %v", vf.Path, err)
		} else {
			scores = s
			source = ReelSourceAnalysis
		}
	}

	// markers usually mark the start of an action: favour the clip that
	// follows each marker
	if times := distinctMarkerTimes(markers, total, 10); len(times) > 0 {
		if scores == nil {
			scores = make([]float64, int(total)+1)
		}
		w := int(math.Ceil(duration))
		for _, t := range times {
			from := int(t) + 1
			for i := from; i < from+w && i < len(scores); i++ {
				scores[i] += 1.5
			}
		}
		source = ReelSourceMarkers
	}

	if scores != nil {
		if clips := pickPeaks(scores, count, duration, total); len(clips) > 0 {
			return &reelPlan{source: source, clips: clips}
		}
	}

	return &reelPlan{source: ReelSourceEven, clips: evenSegments(total, count, duration)}
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}

// evenSegments spreads n clips of segLen seconds across the video, skipping
// the intro and credits.
func evenSegments(total float64, n int, segLen float64) []models.ReelSegment {
	lo := total * 0.08
	hi := total*0.92 - segLen
	if hi < lo {
		lo, hi = 0, math.Max(total-segLen, 0)
	}
	segs := make([]models.ReelSegment, 0, n)
	for i := 0; i < n; i++ {
		var start float64
		if n > 1 {
			start = lo + (hi-lo)*float64(i)/float64(n-1)
		} else {
			start = (lo + hi) / 2
		}
		segs = append(segs, models.ReelSegment{Start: round2(start), End: round2(math.Min(start+segLen, total))})
	}
	return segs
}

// pickPeaks chooses n non-overlapping windows of segLen seconds with the
// highest total score, in chronological order. scores has one value per
// second of video.
func pickPeaks(scores []float64, n int, segLen float64, total float64) []models.ReelSegment {
	w := int(math.Ceil(segLen))
	if w < 1 {
		w = 1
	}
	// skip intros and credits
	lo := int(total * 0.04)
	hi := int(total*0.94) - w
	if hi > len(scores)-w {
		hi = len(scores) - w
	}
	if hi <= lo {
		return nil
	}

	// window sums
	prefix := make([]float64, len(scores)+1)
	for i, s := range scores {
		prefix[i+1] = prefix[i] + s
	}
	type cand struct {
		start int
		score float64
	}
	var cands []cand
	for s := lo; s <= hi; s++ {
		cands = append(cands, cand{s, prefix[s+w] - prefix[s]})
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].score > cands[j].score })

	// keep picks apart so the reel shows different moments; on short videos
	// relax the gap until enough moments fit
	minGap := int(math.Max(float64(w)*3, (total*0.6)/float64(n*2)))
	var picked []int
	for {
		picked = picked[:0]
		for _, c := range cands {
			if len(picked) == n {
				break
			}
			ok := true
			for _, p := range picked {
				if abs(c.start-p) < minGap {
					ok = false
					break
				}
			}
			if ok {
				picked = append(picked, c.start)
			}
		}
		if len(picked) == n || minGap <= w {
			break
		}
		minGap = max(minGap/2, w)
	}
	if len(picked) == 0 {
		return nil
	}
	sort.Ints(picked)

	segs := make([]models.ReelSegment, len(picked))
	for i, p := range picked {
		segs[i] = models.ReelSegment{Start: float64(p), End: round2(float64(p) + segLen)}
	}
	return segs
}

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

// funscriptScores returns movement intensity per second from a funscript.
func funscriptScores(path string, total float64) ([]float64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var script Script
	if err := unmarshalFunscriptData(data, &script); err != nil {
		return nil, err
	}
	if len(script.Actions) < 10 {
		return nil, errors.New("not enough actions")
	}
	sort.SliceStable(script.Actions, func(i, j int) bool { return script.Actions[i].At < script.Actions[j].At })

	scores := make([]float64, int(total)+1)
	for i := 1; i < len(script.Actions); i++ {
		a, b := script.Actions[i-1], script.Actions[i]
		dt := (b.At - a.At) / 1000
		if dt <= 0 {
			continue
		}
		sec := int(b.At / 1000)
		if sec >= 0 && sec < len(scores) {
			scores[sec] += math.Abs(float64(b.Pos-a.Pos)) / dt
		}
	}
	return scores, nil
}

var (
	ptsRe   = regexp.MustCompile(`pts_time:([0-9.]+)`)
	rmsRe   = regexp.MustCompile(`lavfi\.astats\.Overall\.RMS_level=(-?[0-9.]+|-inf|inf|nan)`)
	sceneRe = regexp.MustCompile(`lavfi\.scene_score=([0-9.]+)`)
)

// analysisScores combines audio loudness and visual motion, per second.
func analysisScores(ctx context.Context, vf *models.VideoFile) ([]float64, error) {
	n := int(vf.Duration) + 1
	audio := make([]float64, n)
	motion := make([]float64, n)
	haveAudio := false
	haveMotion := false

	ff := GetInstance().FFMpeg
	if ff == nil {
		return nil, errors.New("ffmpeg not available")
	}

	if vf.AudioCodec != "" {
		// one RMS reading per second of audio
		args := []string{"-hide_banner", "-nostats", "-v", "error", "-i", vf.Path, "-vn", "-sn", "-dn",
			"-af", "aresample=8000,asetnsamples=n=8000,astats=metadata=1:reset=1,ametadata=print:key=lavfi.astats.Overall.RMS_level:file=-",
			"-f", "null", "-"}
		out, err := runFFmpegCapture(ctx, args)
		if err != nil {
			logger.Debugf("[reels] audio analysis: %v", err)
		} else {
			for i := range audio {
				audio[i] = -90
			}
			parseMetadataPairs(out, rmsRe, func(t float64, v string) {
				f, err := strconv.ParseFloat(v, 64)
				if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
					f = -90
				}
				if s := int(t); s >= 0 && s < n {
					audio[s] = math.Max(f, -90)
					haveAudio = true
				}
			})
		}
	}

	// visual change between key frames
	args := []string{"-hide_banner", "-nostats", "-v", "error", "-skip_frame", "nokey", "-i", vf.Path, "-an", "-sn", "-dn",
		"-vf", "scale=96:-2,select='gte(scene,0)',metadata=print:key=lavfi.scene_score:file=-",
		"-fps_mode", "passthrough", "-f", "null", "-"}
	out, err := runFFmpegCapture(ctx, args)
	if err != nil {
		logger.Debugf("[reels] motion analysis: %v", err)
	} else {
		last := -1
		lastVal := 0.0
		parseMetadataPairs(out, sceneRe, func(t float64, v string) {
			f, _ := strconv.ParseFloat(v, 64)
			s := int(t)
			if s < 0 || s >= n {
				return
			}
			// fill the gap since the previous key frame
			for i := last + 1; i < s; i++ {
				motion[i] = (lastVal + f) / 2
			}
			motion[s] = math.Max(motion[s], f)
			last, lastVal = s, f
			haveMotion = true
		})
	}

	if !haveAudio && !haveMotion {
		return nil, errors.New("no audio or motion data")
	}

	za := zscores(audio, haveAudio)
	zm := zscores(motion, haveMotion)
	scores := make([]float64, n)
	for i := range scores {
		switch {
		case haveAudio && haveMotion:
			scores[i] = 0.6*za[i] + 0.4*zm[i]
		case haveAudio:
			scores[i] = za[i]
		default:
			scores[i] = zm[i]
		}
	}
	return scores, nil
}

func zscores(v []float64, valid bool) []float64 {
	ret := make([]float64, len(v))
	if !valid || len(v) == 0 {
		return ret
	}
	mean := 0.0
	for _, x := range v {
		mean += x
	}
	mean /= float64(len(v))
	sd := 0.0
	for _, x := range v {
		sd += (x - mean) * (x - mean)
	}
	sd = math.Sqrt(sd / float64(len(v)))
	if sd == 0 {
		return ret
	}
	for i, x := range v {
		z := (x - mean) / sd
		// cap outliers such as a single loud bang
		ret[i] = math.Max(math.Min(z, 3), -3)
	}
	return ret
}

// parseMetadataPairs reads ffmpeg metadata=print output, where a
// "frame:N pts:N pts_time:T" line is followed by key=value lines.
func parseMetadataPairs(out []byte, valueRe *regexp.Regexp, fn func(t float64, v string)) {
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	cur := -1.0
	for sc.Scan() {
		line := sc.Text()
		if m := ptsRe.FindStringSubmatch(line); m != nil {
			cur, _ = strconv.ParseFloat(m[1], 64)
			continue
		}
		if m := valueRe.FindStringSubmatch(line); m != nil && cur >= 0 {
			fn(cur, m[1])
		}
	}
}

func runFFmpegCapture(ctx context.Context, args []string) ([]byte, error) {
	cmd := GetInstance().FFMpeg.Command(ctx, args)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, lastLines(stderr.String(), 3))
	}
	return stdout.Bytes(), nil
}

func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, " | ")
}

// reelFilterGraph letterboxes one clip onto a blurred vertical canvas. The
// background is blurred at a small size, which costs far less than blurring
// at full size.
func reelFilterGraph(length float64, hasAudio bool) string {
	const fade = 0.3
	fadeOut := math.Max(length-fade, 0)
	var b strings.Builder
	b.WriteString("[0:v]setpts=PTS-STARTPTS,fps=30,split=2[bg][fg];")
	fmt.Fprintf(&b, "[bg]scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,boxblur=6:2,scale=%d:%d[bgb];",
		reelWidth/4, reelHeight/4, reelWidth/4, reelHeight/4, reelWidth, reelHeight)
	fmt.Fprintf(&b, "[fg]scale=%d:%d:force_original_aspect_ratio=decrease[fgs];", reelWidth, reelHeight)
	fmt.Fprintf(&b, "[bgb][fgs]overlay=(W-w)/2:(H-h)/2,setsar=1,format=yuv420p,fade=t=in:st=0:d=%.2f,fade=t=out:st=%.2f:d=%.2f[v]", fade, fadeOut, fade)
	if hasAudio {
		fmt.Fprintf(&b, ";[0:a]asetpts=PTS-STARTPTS,aresample=44100,aformat=channel_layouts=stereo,afade=t=in:st=0:d=%.2f,afade=t=out:st=%.2f:d=%.2f[a]", fade, fadeOut, fade)
	}
	return b.String()
}

func renderReel(ctx context.Context, vf *models.VideoFile, clip models.ReelSegment, out string) error {
	length := clip.End - clip.Start
	if length <= 0 {
		return errors.New("empty clip")
	}
	hasAudio := vf.AudioCodec != ""
	args := []string{"-hide_banner", "-nostats", "-v", "error", "-y",
		"-ss", strconv.FormatFloat(clip.Start, 'f', 2, 64),
		"-t", strconv.FormatFloat(length, 'f', 2, 64),
		"-i", vf.Path,
		"-filter_complex", reelFilterGraph(length, hasAudio),
		"-map", "[v]",
	}
	if hasAudio {
		args = append(args, "-map", "[a]", "-c:a", "aac", "-b:a", "128k")
	} else {
		args = append(args, "-an")
	}
	args = append(args,
		// two encoder threads keep the server responsive while reels are made
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "24", "-profile:v", "high", "-threads", "2",
		"-movflags", "+faststart",
		out)
	if _, err := runFFmpegCapture(ctx, args); err != nil {
		return fmt.Errorf("rendering reel: %w", err)
	}
	return nil
}
