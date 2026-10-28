package ffmpeg

import (
	"context"
	"os"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
)

// Hardware acceleration for one-shot encodes (reels, offline downloads).
//
// Live streaming already decodes, scales and encodes on the GPU where it can
// (see hwDeviceInit and hwMaxResFilter). The helpers here expose the same
// building blocks to jobs that write a file instead of a stream.
//
// There are two levels:
//   - full hardware: decode, scale and encode all stay on the GPU. Fastest, but
//     only possible for a simple linear filter chain and a source the GPU can decode.
//   - hybrid: the GPU decodes, the filter graph runs in system memory, and the
//     result is uploaded to the GPU for encoding. Works with any filter graph.
//
// Callers must always be ready to fall back to software encoding if ffmpeg fails.

// hwDRIDevice returns the render node used for VAAPI.
func hwDRIDevice() string {
	// custom /dev/dri device #6435
	if d := os.Getenv("STASH_HW_DRI_DEVICE"); d != "" {
		return d
	}
	return "/dev/dri/renderD128"
}

// HWBatchCodec returns the hardware H.264 encoder to use for a file output,
// or nil if none was detected. Unlike streaming, VAAPI is acceptable here
// because the output is a regular file rather than a fragmented pipe.
func (f *FFMpeg) HWBatchCodec() *VideoCodec {
	if c := f.hwCodecMP4Compatible(); c != nil {
		return c
	}
	for _, c := range f.getHWCodecSupport() {
		if c == VideoCodecV264 {
			codec := c
			return &codec
		}
	}
	return nil
}

// HWCanFullHW reports whether the file can be decoded, scaled and encoded
// entirely on the GPU with the codec. It runs a short ffmpeg probe.
func (f *FFMpeg) HWCanFullHW(ctx context.Context, codec VideoCodec, vf *models.VideoFile, height int) bool {
	return f.hwCanFullHWTranscode(ctx, codec, vf, height)
}

// HWFullInputArgs returns the input arguments (placed before -i) for full hardware transcoding.
func (f *FFMpeg) HWFullInputArgs(codec VideoCodec) Args {
	return f.hwDeviceInit(nil, codec, true)
}

// HWFullFilter returns the video filter for full hardware transcoding,
// downscaling to at most height (0 keeps the size).
func (f *FFMpeg) HWFullFilter(codec VideoCodec, vf *models.VideoFile, height int) VideoFilter {
	return f.hwMaxResFilter(codec, vf, height, true)
}

// HWHybridInputArgs returns the input arguments (placed before -i) for GPU
// decoding with frames copied back to system memory, plus the device setup the
// encoder needs. ok is false when the codec has no hybrid mode.
func (f *FFMpeg) HWHybridInputArgs(codec VideoCodec) (args Args, ok bool) {
	switch codec {
	case VideoCodecV264:
		dev := hwDRIDevice()
		return Args{"-hwaccel", "vaapi", "-hwaccel_device", dev, "-vaapi_device", dev}, true
	case VideoCodecI264, VideoCodecI264C:
		return Args{"-init_hw_device", "qsv=hw", "-filter_hw_device", "hw", "-hwaccel", "qsv", "-hwaccel_device", "hw"}, true
	case VideoCodecN264, VideoCodecN264H:
		return Args{"-hwaccel_device", "0", "-hwaccel", "cuda"}, true
	}
	return nil, false
}

// HWHybridFilterTail returns the filter(s) that must end a system-memory
// filter graph so the frames can be fed to the hardware encoder. The result
// has no leading or trailing comma.
func (f *FFMpeg) HWHybridFilterTail(codec VideoCodec) (string, bool) {
	switch codec {
	case VideoCodecV264:
		return "format=nv12,hwupload", true
	case VideoCodecI264, VideoCodecI264C:
		return "format=nv12,hwupload=extra_hw_frames=64,format=qsv", true
	case VideoCodecN264, VideoCodecN264H:
		return "format=nv12,hwupload_cuda", true
	}
	return "", false
}

// HWQualityArgs returns encoder arguments that map an x264-style CRF to the
// hardware encoder's own quality setting. They are meant to follow CodecInit,
// where the last occurrence of an option wins.
func HWQualityArgs(codec VideoCodec, crf int) Args {
	if crf <= 0 {
		return nil
	}
	q := strconv.Itoa(crf)
	switch codec {
	case VideoCodecN264, VideoCodecN264H:
		return Args{"-cq", q}
	case VideoCodecI264:
		return Args{"-global_quality", q}
	case VideoCodecI264C:
		return Args{"-q", q}
	case VideoCodecV264:
		return Args{"-qp", q}
	}
	return nil
}
