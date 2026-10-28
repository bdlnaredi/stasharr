package manager

// Offline downloads for the phone app. "Original" downloads the library file
// as it is. The smaller qualities are converted to an H.264 MP4 once, kept
// for a week after their last download, and then deleted.

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stashapp/stash/pkg/ffmpeg"
	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
)

// OfflineQuality is the size a download is converted to.
type OfflineQuality struct {
	ID     string
	Label  string
	Height int // 0 keeps the original file
	CRF    int
	Audio  string
}

// OfflineQualities lists the download sizes, best first.
var OfflineQualities = []OfflineQuality{
	{ID: "original", Label: "Original file"},
	{ID: "1080", Label: "1080p", Height: 1080, CRF: 23, Audio: "160k"},
	{ID: "720", Label: "720p", Height: 720, CRF: 24, Audio: "128k"},
	{ID: "480", Label: "480p (smallest)", Height: 480, CRF: 26, Audio: "96k"},
}

// FindOfflineQuality returns the quality with the id.
func FindOfflineQuality(id string) (OfflineQuality, bool) {
	for _, q := range OfflineQualities {
		if q.ID == id {
			return q, true
		}
	}
	return OfflineQuality{}, false
}

// Offline download states.
const (
	OfflineQueued     = "QUEUED"
	OfflineConverting = "CONVERTING"
	OfflineReady      = "READY"
	OfflineFailed     = "FAILED"
)

// OfflineState describes a download being prepared.
type OfflineState struct {
	Status   string
	Progress float64
	Size     int64
	Error    string
}

type offlineJob struct {
	sceneID  int
	quality  OfflineQuality
	input    string
	duration float64
	vf       *models.VideoFile
}

// how long converted files are kept after their last download
const offlineKeep = 7 * 24 * time.Hour

type offlineManager struct {
	mu     sync.Mutex
	states map[string]*OfflineState
	queue  chan offlineJob
	once   sync.Once
}

var offline = &offlineManager{
	states: map[string]*OfflineState{},
	queue:  make(chan offlineJob, 200),
}

func offlineKey(sceneID int, quality string) string {
	return strconv.Itoa(sceneID) + "_" + quality
}

func offlineDir() string {
	return filepath.Join(GetInstance().Config.GetGeneratedPath(), "offline")
}

// OfflineFilePath is where the converted file of a scene is kept.
func OfflineFilePath(sceneID int, quality string) string {
	return filepath.Join(offlineDir(), offlineKey(sceneID, quality)+".mp4")
}

func (m *offlineManager) start() {
	m.once.Do(func() {
		go m.worker()
		go m.cleanLoop()
	})
}

// OfflineStatus reports whether the scene's file in this quality is ready.
// For a file not asked for yet it returns nil.
func OfflineStatus(sceneID int, quality string) *OfflineState {
	q, ok := FindOfflineQuality(quality)
	if !ok {
		return nil
	}
	if q.Height == 0 {
		return &OfflineState{Status: OfflineReady, Progress: 1}
	}
	offline.mu.Lock()
	defer offline.mu.Unlock()
	if st, ok := offline.states[offlineKey(sceneID, quality)]; ok {
		cp := *st
		return &cp
	}
	if info, err := os.Stat(OfflineFilePath(sceneID, quality)); err == nil {
		return &OfflineState{Status: OfflineReady, Progress: 1, Size: info.Size()}
	}
	return nil
}

// PrepareOffline starts converting the scene's file for download, unless it
// is ready or already being converted.
func PrepareOffline(scene *models.Scene, quality string) (*OfflineState, error) {
	q, ok := FindOfflineQuality(quality)
	if !ok {
		return nil, fmt.Errorf("unknown quality %q", quality)
	}
	f := scene.Files.Primary()
	if f == nil {
		return nil, errors.New("the scene has no file")
	}
	if q.Height == 0 {
		return &OfflineState{Status: OfflineReady, Progress: 1, Size: f.Size}, nil
	}
	if GetInstance().FFMpeg == nil {
		return nil, errors.New("ffmpeg is not available on the server")
	}

	offline.start()
	key := offlineKey(scene.ID, quality)

	offline.mu.Lock()
	defer offline.mu.Unlock()
	if st, ok := offline.states[key]; ok && st.Status != OfflineFailed {
		cp := *st
		return &cp, nil
	}
	if info, err := os.Stat(OfflineFilePath(scene.ID, quality)); err == nil {
		return &OfflineState{Status: OfflineReady, Progress: 1, Size: info.Size()}, nil
	}

	st := &OfflineState{Status: OfflineQueued}
	select {
	case offline.queue <- offlineJob{sceneID: scene.ID, quality: q, input: f.Path, duration: f.Duration, vf: f}:
	default:
		return nil, errors.New("too many downloads are being prepared; try again later")
	}
	offline.states[key] = st
	cp := *st
	return &cp, nil
}

func (m *offlineManager) set(key string, fn func(st *OfflineState)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st, ok := m.states[key]
	if !ok {
		st = &OfflineState{}
		m.states[key] = st
	}
	fn(st)
}

func (m *offlineManager) worker() {
	for j := range m.queue {
		key := offlineKey(j.sceneID, j.quality.ID)
		m.set(key, func(st *OfflineState) { st.Status = OfflineConverting })
		size, err := convertOffline(j, func(p float64) {
			m.set(key, func(st *OfflineState) { st.Progress = p })
		})
		if err != nil {
			logger.Errorf("[offline] scene %d %s: %v", j.sceneID, j.quality.ID, err)
			m.set(key, func(st *OfflineState) {
				st.Status = OfflineFailed
				st.Error = err.Error()
			})
			continue
		}
		// the file on disk is now the source of truth
		m.mu.Lock()
		delete(m.states, key)
		m.mu.Unlock()
		logger.Infof("[offline] scene %d ready in %s (%s)", j.sceneID, j.quality.Label, humanSize(size))
	}
}

func convertOffline(j offlineJob, progress func(float64)) (int64, error) {
	ff := GetInstance().FFMpeg
	if err := fsutil.EnsureDir(offlineDir()); err != nil {
		return 0, err
	}
	out := OfflineFilePath(j.sceneID, j.quality.ID)
	tmp := out + ".part.mp4"
	_ = os.Remove(tmp)

	// Try the GPU first (full hardware, then hybrid), and fall back to
	// software if ffmpeg cannot do it.
	done := false
	for _, mode := range offlineHWModes(j) {
		args := offlineArgs(j, tmp, mode)
		if err := runOffline(ff, args, j.duration, progress); err != nil {
			logger.Warnf("[offline] scene %d: hardware encoding (%s) failed, trying another way: %v", j.sceneID, mode.name, err)
			_ = os.Remove(tmp)
			continue
		}
		done = true
		break
	}
	if !done {
		if err := runOffline(ff, offlineArgs(j, tmp, offlineMode{}), j.duration, progress); err != nil {
			_ = os.Remove(tmp)
			return 0, err
		}
	}

	if err := os.Rename(tmp, out); err != nil {
		return 0, err
	}
	info, err := os.Stat(out)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// offlineMode describes how a conversion uses the GPU. The zero value is software only.
type offlineMode struct {
	name   string
	codec  ffmpeg.VideoCodec
	full   bool // decode, scale and encode on the GPU
	hybrid bool // decode and encode on the GPU, scale on the CPU
}

// offlineHWModes lists the hardware modes to try, best first. It is empty
// when hardware acceleration is off or no hardware encoder was found.
func offlineHWModes(j offlineJob) []offlineMode {
	m := GetInstance()
	if m.FFMpeg == nil || !m.Config.GetTranscodeHardwareAcceleration() {
		return nil
	}
	c := m.FFMpeg.HWBatchCodec()
	if c == nil {
		return nil
	}

	var modes []offlineMode
	if j.vf != nil && j.vf.BaseFile != nil && m.FFMpeg.HWCanFullHW(context.Background(), *c, j.vf, j.quality.Height) {
		modes = append(modes, offlineMode{name: "full " + c.CodeName, codec: *c, full: true})
	}
	if _, ok := m.FFMpeg.HWHybridInputArgs(*c); ok {
		modes = append(modes, offlineMode{name: "hybrid " + c.CodeName, codec: *c, hybrid: true})
	}
	return modes
}

func offlineArgs(j offlineJob, tmp string, mode offlineMode) []string {
	ff := GetInstance().FFMpeg
	args := []string{"-hide_banner", "-nostats", "-v", "error", "-y"}

	scale := fmt.Sprintf("scale=-2:'min(%d,ih)'", j.quality.Height)
	vfilter := scale
	switch {
	case mode.full:
		args = append(args, ff.HWFullInputArgs(mode.codec)...)
		vfilter = string(ff.HWFullFilter(mode.codec, j.vf, j.quality.Height))
	case mode.hybrid:
		in, _ := ff.HWHybridInputArgs(mode.codec)
		args = append(args, in...)
		if tail, ok := ff.HWHybridFilterTail(mode.codec); ok {
			vfilter = scale + "," + tail
		}
	}

	args = append(args, "-i", j.input, "-map", "0:v:0", "-map", "0:a:0?", "-sn", "-dn")
	if vfilter != "" {
		args = append(args, "-vf", vfilter)
	}

	if mode.full || mode.hybrid {
		args = append(args, ffmpeg.CodecInit(mode.codec)...)
		args = append(args, ffmpeg.HWQualityArgs(mode.codec, j.quality.CRF)...)
		args = append(args, "-profile:v", "high")
	} else {
		args = append(args,
			"-c:v", "libx264", "-preset", "veryfast", "-crf", strconv.Itoa(j.quality.CRF),
			"-profile:v", "high", "-pix_fmt", "yuv420p")
	}

	args = append(args,
		"-c:a", "aac", "-b:a", j.quality.Audio, "-ac", "2",
		"-movflags", "+faststart",
		"-progress", "pipe:1",
		tmp)
	return args
}

// runOffline runs ffmpeg, reporting progress from its -progress output.
func runOffline(ff *ffmpeg.FFMpeg, args []string, duration float64, progress func(float64)) error {
	cmd := ff.Command(context.Background(), args)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		line := sc.Text()
		if v, ok := strings.CutPrefix(line, "out_time_us="); ok && duration > 0 {
			us, err := strconv.ParseFloat(v, 64)
			if err == nil {
				progress(math.Min(0.99, us/1e6/duration))
			}
		}
	}
	if err := cmd.Wait(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if len(msg) > 300 {
			msg = msg[len(msg)-300:]
		}
		if msg == "" {
			msg = err.Error()
		}
		return errors.New(msg)
	}
	return nil
}

// TouchOffline marks a converted file as used, so it is kept longer.
func TouchOffline(path string) {
	now := time.Now()
	_ = os.Chtimes(path, now, now)
}

func (m *offlineManager) cleanLoop() {
	for {
		entries, err := os.ReadDir(offlineDir())
		if err == nil {
			for _, e := range entries {
				info, err := e.Info()
				if err != nil || e.IsDir() {
					continue
				}
				if time.Since(info.ModTime()) > offlineKeep {
					p := filepath.Join(offlineDir(), e.Name())
					if err := os.Remove(p); err == nil {
						logger.Debugf("[offline] removed old download %s", e.Name())
					}
				}
			}
		}
		time.Sleep(6 * time.Hour)
	}
}
