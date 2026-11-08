package manager

import (
	"math"
	"testing"
)

func TestQuantize(t *testing.T) {
	v := []float64{0.5, -0.25, 0.1, 0}
	q, scale := quantize(v)
	for i, x := range v {
		got := float64(int8(q[i])) * scale
		if math.Abs(got-x) > scale {
			t.Errorf("%d: got %v want %v", i, got, x)
		}
	}
	if int8(q[0]) != 127 {
		t.Errorf("largest value should use the full range, got %d", int8(q[0]))
	}
}
