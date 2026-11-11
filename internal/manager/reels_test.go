package manager

import (
	"testing"

	"github.com/stashapp/stash/pkg/models"
)

func TestPickPeaksFindsLoudPart(t *testing.T) {
	scores := make([]float64, 120)
	for i := 60; i < 64; i++ {
		scores[i] = 10
	}
	segs := pickPeaks(scores, 5, 3, 120)
	if len(segs) != 5 {
		t.Fatalf("want 5 segments, got %d", len(segs))
	}
	found := false
	for i, s := range segs {
		if i > 0 && s.Start < segs[i-1].End {
			t.Errorf("segments overlap or are out of order: %v", segs)
		}
		if s.Start >= 58 && s.Start <= 62 {
			found = true
		}
	}
	if !found {
		t.Errorf("loud part not chosen: %v", segs)
	}
}

func TestPickPeaksShortVideo(t *testing.T) {
	scores := make([]float64, 40)
	segs := pickPeaks(scores, 5, 3, 40)
	if len(segs) != 5 {
		t.Fatalf("want 5 segments on a short video, got %d (%v)", len(segs), segs)
	}
}

func TestPickPeaksContinuousClips(t *testing.T) {
	// a 30 minute scene with three busy parts
	scores := make([]float64, 1800)
	for _, start := range []int{300, 900, 1500} {
		for i := start; i < start+20; i++ {
			scores[i] = 5
		}
	}
	clips := pickPeaks(scores, 3, 15, 1800)
	if len(clips) != 3 {
		t.Fatalf("want 3 clips, got %v", clips)
	}
	for i, c := range clips {
		if l := c.End - c.Start; l < 14.9 || l > 15.1 {
			t.Errorf("clip %d is %vs long, want 15s", i, l)
		}
		if i > 0 && c.Start < clips[i-1].End {
			t.Errorf("clips overlap: %v", clips)
		}
	}
	for i, want := range []float64{300, 900, 1500} {
		if clips[i].Start < want-1 || clips[i].Start > want+6 {
			t.Errorf("clip %d starts at %v, want about %v", i, clips[i].Start, want)
		}
	}
}

func TestDistinctMarkerTimes(t *testing.T) {
	markers := []*models.SceneMarker{{Seconds: 600}, {Seconds: 10}, {Seconds: 12}, {Seconds: 300}, {Seconds: 899.5}}
	got := distinctMarkerTimes(markers, 900, 10)
	// 10 and 12 are merged, and a marker in the last second is dropped
	want := []float64{10, 300, 600}
	if len(got) != len(want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("want %v, got %v", want, got)
		}
	}
}

func TestEvenSegments(t *testing.T) {
	clips := evenSegments(1000, 3, 15)
	if len(clips) != 3 {
		t.Fatalf("want 3 clips, got %v", clips)
	}
	for i, c := range clips {
		if l := c.End - c.Start; l < 14.99 || l > 15.01 {
			t.Errorf("clip %d is not 15s: %v", i, c)
		}
		if c.Start < 0 || c.End > 1000 {
			t.Errorf("clip %d outside the video: %v", i, c)
		}
	}
	if one := evenSegments(20, 1, 15); len(one) != 1 || one[0].End > 20 {
		t.Errorf("short video: %v", one)
	}
}

func TestCollapsePaths(t *testing.T) {
	got := collapsePaths([]string{"/a/b", "/a", "/c", "/a/b/c"})
	if len(got) != 2 || got[0] != "/a" || got[1] != "/c" {
		t.Errorf("got %v", got)
	}
}

func TestPickVariedPeaks(t *testing.T) {
	// 300 seconds: act 0 strong at 50-80 and 200-230, act 1 a bit weaker at 120-150
	n := 300
	scores := make([]float64, n)
	act := make([]int, n)
	for i := range act {
		act[i] = -1
	}
	for i := 50; i < 80; i++ {
		scores[i], act[i] = 3, 0
	}
	for i := 200; i < 230; i++ {
		scores[i], act[i] = 3, 0
	}
	for i := 120; i < 150; i++ {
		scores[i], act[i] = 2.5, 1
	}
	got := pickVariedPeaks(scores, act, 2, 15, float64(n))
	if len(got) != 2 {
		t.Fatalf("got %d reels, want 2", len(got))
	}
	// the second reel should show the other act, not the second act-0 peak
	acts := map[int]bool{}
	for _, g := range got {
		acts[act[int(g.Start)+5]] = true
	}
	if !acts[0] || !acts[1] {
		t.Errorf("want one reel of each act, got %+v", got)
	}
	for i := 1; i < len(got); i++ {
		if got[i].Start < got[i-1].End {
			t.Errorf("reels overlap: %+v", got)
		}
	}
}
