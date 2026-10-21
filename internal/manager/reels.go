package manager

// Automatic short vertical reels, built from the most interesting moments
// of a scene.
//
// Moments are chosen, in order of preference, from:
//   - MARKERS:  the scene's markers (from you, or from ThePornDB / StashDB)
//   - HEATMAP:  the intensity of an interactive funscript next to the video
//   - ANALYSIS: loudness of the audio track combined with on-screen motion,
//     measured by ffmpeg
//
// The chosen moments are cut into equal segments, letterboxed onto a
// blurred 9:16 background and joined into a single MP4.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
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
	SceneIDs  []int
	Duration  float64
	Overwrite bool
}

func (j *ReelJob) Execute(ctx context.Context, progress *job.Progress) error {
	mgr := GetInstance()
	r := mgr.Repository

	duration := j.Duration
	if duration <= 0 {
		duration = mgr.Config.GetReelDuration()
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
		progress.ExecuteTask(fmt.Sprintf("Making reel for scene %d", id), func() {
			if _, err := GenerateReel(ctx, id, duration, j.Overwrite); err != nil {
				if errors.Is(err, errReelExists) {
					return
				}
				logger.Errorf("[reels] scene %d: %v", id, err)
			}
		})
		progress.Increment()
	}
	return nil
}

var errReelExists = errors.New("scene already has a reel")

type reelPlan struct {
	source   string
	segments []models.ReelSegment
}

// GenerateReel makes a reel for one scene and stores it.
func GenerateReel(ctx context.Context, sceneID int, duration float64, overwrite bool) (*models.Reel, error) {
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

	if len(old) > 0 && !overwrite {
		return nil, errReelExists
	}

	if vf.Duration <= 0 {
		return nil, errors.New("video duration unknown, scan the file first")
	}

	plan, err := planReel(ctx, vf, markers, duration)
	if err != nil {
		return nil, err
	}

	outDir := mgr.Paths.Generated.Reels
	if err := fsutil.EnsureDir(outDir); err != nil {
		return nil, err
	}
	out := filepath.Join(outDir, fmt.Sprintf("%d_%d.mp4", sceneID, time.Now().Unix()))

	if err := renderReel(ctx, vf, plan.segments, out); err != nil {
		_ = os.Remove(out)
		return nil, err
	}

	total := 0.0
	for _, s := range plan.segments {
		total += s.End - s.Start
	}

	reel := &models.Reel{
		SceneID:  sceneID,
		Path:     out,
		Duration: total,
		Width:    reelWidth,
		Height:   reelHeight,
		Source:   plan.source,
		Segments: plan.segments,
	}

	if err := r.WithTxn(ctx, func(ctx context.Context) error {
		for _, o := range old {
			if err := r.Reel.Destroy(ctx, o.ID); err != nil {
				return err
			}
		}
		return r.Reel.Create(ctx, reel)
	}); err != nil {
		_ = os.Remove(out)
		return nil, err
	}

	for _, o := range old {
		_ = os.Remove(o.Path)
	}

	logger.Infof("[reels] made %.0fs reel for %q from %s", total, scene.GetTitle(), strings.ToLower(plan.source))
	return reel, nil
}

// segmentCount returns how many cuts make up a reel of the given length.
func segmentCount(duration float64) int {
	n := int(math.Round(duration / 3))
	if n < 3 {
		n = 3
	}
	if n > 6 {
		n = 6
	}
	return n
}

func planReel(ctx context.Context, vf *models.VideoFile, markers []*models.SceneMarker, duration float64) (*reelPlan, error) {
	total := vf.Duration

	// short videos: use them whole
	if total <= duration+1 {
		return &reelPlan{source: ReelSourceEven, segments: []models.ReelSegment{{Start: 0, End: total}}}, nil
	}

	if segs := segmentsFromMarkers(markers, total, duration); len(segs) > 0 {
		return &reelPlan{source: ReelSourceMarkers, segments: segs}, nil
	}

	n := segmentCount(duration)
	segLen := duration / float64(n)

	if vf.Interactive {
		if scores, err := funscriptScores(video.GetFunscriptPath(vf.Path), total); err == nil && len(scores) > 0 {
			if segs := pickPeaks(scores, n, segLen, total); len(segs) > 0 {
				return &reelPlan{source: ReelSourceHeatmap, segments: segs}, nil
			}
		}
	}

	scores, err := analysisScores(ctx, vf)
	if err != nil {
		logger.Warnf("[reels] analysis of %s failed, using evenly spaced moments: %v", vf.Path, err)
	} else if segs := pickPeaks(scores, n, segLen, total); len(segs) > 0 {
		return &reelPlan{source: ReelSourceAnalysis, segments: segs}, nil
	}

	return &reelPlan{source: ReelSourceEven, segments: evenSegments(total, n, segLen)}, nil
}

// segmentsFromMarkers uses markers spread across the scene. At least two
// distinct markers are needed.
func segmentsFromMarkers(markers []*models.SceneMarker, total, duration float64) []models.ReelSegment {
	var times []float64
	for _, m := range markers {
		if m.Seconds >= 0 && m.Seconds < total-1 {
			times = append(times, m.Seconds)
		}
	}
	sort.Float64s(times)

	// drop markers that are close together
	var distinct []float64
	for _, t := range times {
		if len(distinct) == 0 || t-distinct[len(distinct)-1] >= 10 {
			distinct = append(distinct, t)
		}
	}
	if len(distinct) < 2 {
		return nil
	}

	n := segmentCount(duration)
	if len(distinct) > n {
		// spread the choice evenly across the markers
		picked := make([]float64, 0, n)
		for i := 0; i < n; i++ {
			picked = append(picked, distinct[i*len(distinct)/n])
		}
		distinct = picked
	}
	n = len(distinct)
	segLen := duration / float64(n)

	segs := make([]models.ReelSegment, 0, n)
	for _, t := range distinct {
		// markers usually mark the start of an action: skip the first moment
		start := t + 1
		if start+segLen > total {
			start = total - segLen
		}
		if start < 0 {
			start = 0
		}
		segs = append(segs, models.ReelSegment{Start: round2(start), End: round2(start + segLen)})
	}
	return segs
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}

func evenSegments(total float64, n int, segLen float64) []models.ReelSegment {
	lo := total * 0.08
	hi := total*0.92 - segLen
	segs := make([]models.ReelSegment, 0, n)
	for i := 0; i < n; i++ {
		start := lo + (hi-lo)*float64(i)/float64(max(n-1, 1))
		segs = append(segs, models.ReelSegment{Start: round2(start), End: round2(start + segLen)})
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

// reelFilterGraph builds the ffmpeg filter graph joining the segments onto
// a blurred vertical canvas.
func reelFilterGraph(segs []models.ReelSegment, hasAudio bool) string {
	var b strings.Builder
	var concatInputs strings.Builder
	const fade = 0.2
	for i, s := range segs {
		l := s.End - s.Start
		fadeOut := math.Max(l-fade, 0)
		fmt.Fprintf(&b, "[%d:v]setpts=PTS-STARTPTS,split=2[bg%d][fg%d];", i, i, i)
		fmt.Fprintf(&b, "[bg%d]scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,boxblur=20:2[bgb%d];", i, reelWidth, reelHeight, reelWidth, reelHeight, i)
		fmt.Fprintf(&b, "[fg%d]scale=%d:%d:force_original_aspect_ratio=decrease[fgs%d];", i, reelWidth, reelHeight, i)
		fmt.Fprintf(&b, "[bgb%d][fgs%d]overlay=(W-w)/2:(H-h)/2,setsar=1,fps=30,format=yuv420p,fade=t=in:st=0:d=%.2f,fade=t=out:st=%.2f:d=%.2f[v%d];", i, i, fade, fadeOut, fade, i)
		if hasAudio {
			fmt.Fprintf(&b, "[%d:a]asetpts=PTS-STARTPTS,aresample=44100,aformat=channel_layouts=stereo,afade=t=in:st=0:d=%.2f,afade=t=out:st=%.2f:d=%.2f[a%d];", i, fade, fadeOut, fade, i)
		} else {
			fmt.Fprintf(&b, "anullsrc=r=44100:cl=stereo,atrim=0:%.2f[a%d];", l, i)
		}
		fmt.Fprintf(&concatInputs, "[v%d][a%d]", i, i)
	}
	fmt.Fprintf(&b, "%sconcat=n=%d:v=1:a=1[v][a]", concatInputs.String(), len(segs))
	return b.String()
}

func renderReel(ctx context.Context, vf *models.VideoFile, segs []models.ReelSegment, out string) error {
	if len(segs) == 0 {
		return errors.New("no moments to use")
	}
	args := []string{"-hide_banner", "-nostats", "-v", "error", "-y"}
	for _, s := range segs {
		args = append(args,
			"-ss", strconv.FormatFloat(s.Start, 'f', 2, 64),
			"-t", strconv.FormatFloat(s.End-s.Start, 'f', 2, 64),
			"-i", vf.Path)
	}
	args = append(args,
		"-filter_complex", reelFilterGraph(segs, vf.AudioCodec != ""),
		"-map", "[v]", "-map", "[a]",
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "23", "-profile:v", "high",
		"-c:a", "aac", "-b:a", "128k",
		"-movflags", "+faststart",
		out)
	_, err := runFFmpegCapture(ctx, args)
	if err != nil {
		return fmt.Errorf("rendering reel: %w", err)
	}
	return nil
}

// reelJSON is used for logging plans.
func reelJSON(segs []models.ReelSegment) string {
	b, _ := json.Marshal(segs)
	return string(b)
}
