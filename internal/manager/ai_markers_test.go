package manager

import (
	"bytes"
	"testing"
)

func TestSplitJPEGs(t *testing.T) {
	a := []byte{0xFF, 0xD8, 1, 2, 0xFF, 0x00, 3, 0xFF, 0xD9}
	b := []byte{0xFF, 0xD8, 9, 0xFF, 0xD9}
	got := splitJPEGs(append(append([]byte{}, a...), b...))
	if len(got) != 2 || !bytes.Equal(got[0], a) || !bytes.Equal(got[1], b) {
		t.Fatalf("got %v", got)
	}
}

func TestPlanAIMarkers(t *testing.T) {
	labels := []AILabel{
		{Name: "Kissing", Marker: true},
		{Name: "Cowgirl", Marker: true},
		{Name: "Talking", Marker: false},
	}
	row := func(k, c, talk float64) []float64 { return []float64{k, c, talk} }
	var scores [][]float64
	// 0-16s talking, 16-36s kissing (one odd frame), 36-64s cowgirl, 2 frames kissing at the end
	for i := 0; i < 4; i++ {
		scores = append(scores, row(0.1, 0.1, 0.8))
	}
	for i := 0; i < 5; i++ {
		if i == 2 {
			scores = append(scores, row(0.2, 0.5, 0.3))
			continue
		}
		scores = append(scores, row(0.8, 0.1, 0.1))
	}
	for i := 0; i < 7; i++ {
		scores = append(scores, row(0.1, 0.85, 0.05))
	}
	scores = append(scores, row(0.9, 0.05, 0.05), row(0.9, 0.05, 0.05))

	got := planAIMarkers(scores, labels, 4, 0.35, 12)
	if len(got) != 2 {
		t.Fatalf("expected 2 segments, got %+v", got)
	}
	if got[0].Label != "Kissing" || got[0].Start != 16 || got[0].End != 36 {
		t.Errorf("first segment %+v", got[0])
	}
	if got[1].Label != "Cowgirl" || got[1].Start != 36 {
		t.Errorf("second segment %+v", got[1])
	}
	if got[0].Score <= 0.35 || got[0].Score > 1 {
		t.Errorf("score %v", got[0].Score)
	}
}

func TestPlanAIMarkersNotConfident(t *testing.T) {
	labels := []AILabel{{Name: "A", Marker: true}, {Name: "B", Marker: true}}
	var scores [][]float64
	for i := 0; i < 10; i++ {
		scores = append(scores, []float64{0.3, 0.3})
	}
	if got := planAIMarkers(scores, labels, 4, 0.35, 8); len(got) != 0 {
		t.Fatalf("expected none, got %+v", got)
	}
}
