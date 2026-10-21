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

func TestSegmentsFromMarkers(t *testing.T) {
	markers := []*models.SceneMarker{{Seconds: 10}, {Seconds: 12}, {Seconds: 300}, {Seconds: 600}}
	segs := segmentsFromMarkers(markers, 900, 15)
	// 10 and 12 are merged, leaving three distinct moments
	if len(segs) != 3 {
		t.Fatalf("want 3 segments, got %v", segs)
	}
	total := 0.0
	for _, s := range segs {
		total += s.End - s.Start
	}
	if total < 14.9 || total > 15.1 {
		t.Errorf("want 15s in total, got %v", total)
	}

	if segmentsFromMarkers(markers[:1], 900, 15) != nil {
		t.Error("one marker should not be enough")
	}
}

func TestCollapsePaths(t *testing.T) {
	got := collapsePaths([]string{"/a/b", "/a", "/c", "/a/b/c"})
	if len(got) != 2 || got[0] != "/a" || got[1] != "/c" {
		t.Errorf("got %v", got)
	}
}
