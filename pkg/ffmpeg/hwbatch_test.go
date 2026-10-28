package ffmpeg

import (
	"reflect"
	"testing"
)

func TestHWQualityArgs(t *testing.T) {
	tests := []struct {
		name  string
		codec VideoCodec
		crf   int
		want  Args
	}{
		{"nvenc", VideoCodecN264, 24, Args{"-cq", "24"}},
		{"qsv", VideoCodecI264, 23, Args{"-global_quality", "23"}},
		{"qsv compat", VideoCodecI264C, 23, Args{"-q", "23"}},
		{"vaapi", VideoCodecV264, 26, Args{"-qp", "26"}},
		{"zero crf keeps encoder default", VideoCodecV264, 0, nil},
		{"software codec", VideoCodecLibX264, 24, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HWQualityArgs(tt.codec, tt.crf); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HWQualityArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHWHybrid(t *testing.T) {
	f := &FFMpeg{}

	for _, c := range []VideoCodec{VideoCodecV264, VideoCodecI264, VideoCodecI264C, VideoCodecN264, VideoCodecN264H} {
		if _, ok := f.HWHybridInputArgs(c); !ok {
			t.Errorf("%s: expected hybrid input args", c.CodeName)
		}
		if tail, ok := f.HWHybridFilterTail(c); !ok || tail == "" {
			t.Errorf("%s: expected hybrid filter tail", c.CodeName)
		}
	}

	// codecs without a hybrid mode must report so, so callers fall back to software
	for _, c := range []VideoCodec{VideoCodecLibX264, VideoCodecRK264, VideoCodecM264} {
		if _, ok := f.HWHybridInputArgs(c); ok {
			t.Errorf("%s: unexpected hybrid input args", c.CodeName)
		}
		if _, ok := f.HWHybridFilterTail(c); ok {
			t.Errorf("%s: unexpected hybrid filter tail", c.CodeName)
		}
	}
}

func TestHWBatchCodecPrefersMP4ThenVAAPI(t *testing.T) {
	f := &FFMpeg{}

	if f.HWBatchCodec() != nil {
		t.Error("expected no codec when none are supported")
	}

	f.hwCodecSupport = []VideoCodec{VideoCodecV264}
	if c := f.HWBatchCodec(); c == nil || *c != VideoCodecV264 {
		t.Errorf("expected VAAPI to be used for file output, got %v", c)
	}

	f.hwCodecSupport = []VideoCodec{VideoCodecV264, VideoCodecI264}
	if c := f.HWBatchCodec(); c == nil || *c != VideoCodecI264 {
		t.Errorf("expected QSV to be preferred over VAAPI, got %v", c)
	}
}
