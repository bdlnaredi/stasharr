package generate

import (
	"context"
	"os"

	"github.com/stashapp/stash/pkg/ffmpeg"
	"github.com/stashapp/stash/pkg/ffmpeg/transcoder"
	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/logger"
)

// transcodeCRF is the x264-style quality of generated transcodes. Hardware
// encoders map it to their own quality setting.
const transcodeCRF = 23

// hwAccelConfig is implemented by configs that expose the hardware
// acceleration setting. It is optional so other FFMpegConfig implementations
// keep working, and simply never use the GPU.
type hwAccelConfig interface {
	GetTranscodeHardwareAcceleration() bool
}

// hwTranscodeMode describes how a transcode uses the GPU.
type hwTranscodeMode struct {
	name   string
	codec  ffmpeg.VideoCodec
	full   bool // decode, scale and encode on the GPU
	hybrid bool // decode and encode on the GPU, scale on the CPU
}

// hwTranscodeModes lists the hardware modes to try, best first. It is empty
// when hardware acceleration is off or no hardware encoder was detected.
func (g Generator) hwTranscodeModes(ctx context.Context, options TranscodeOptions) []hwTranscodeMode {
	cfg, ok := g.FFMpegConfig.(hwAccelConfig)
	if !ok || !cfg.GetTranscodeHardwareAcceleration() || g.Encoder == nil {
		return nil
	}

	c := g.Encoder.HWBatchCodec()
	if c == nil {
		return nil
	}

	var modes []hwTranscodeMode
	if options.VideoFile != nil && options.VideoFile.BaseFile != nil &&
		g.Encoder.HWCanFullHW(ctx, *c, options.VideoFile, options.Height) {
		modes = append(modes, hwTranscodeMode{name: "full " + c.CodeName, codec: *c, full: true})
	}
	if _, ok := g.Encoder.HWHybridInputArgs(*c); ok {
		modes = append(modes, hwTranscodeMode{name: "hybrid " + c.CodeName, codec: *c, hybrid: true})
	}
	return modes
}

// hwTranscodeArgs builds the ffmpeg arguments for a hardware transcode.
func (g Generator) hwTranscodeArgs(input, tmpFn string, options TranscodeOptions, withAudio bool, mode hwTranscodeMode) ffmpeg.Args {
	var inputArgs ffmpeg.Args
	var filter ffmpeg.VideoFilter

	switch {
	case mode.full:
		inputArgs = g.Encoder.HWFullInputArgs(mode.codec)
		filter = g.Encoder.HWFullFilter(mode.codec, options.VideoFile, options.Height)
	case mode.hybrid:
		inputArgs, _ = g.Encoder.HWHybridInputArgs(mode.codec)
		if options.Width != 0 && options.Height != 0 {
			filter = filter.ScaleDimensions(options.Width, options.Height)
		}
		if tail, ok := g.Encoder.HWHybridFilterTail(mode.codec); ok {
			filter = filter.Append(tail)
		}
	}

	var videoArgs ffmpeg.Args
	videoArgs = videoArgs.VideoFilter(filter)
	videoArgs = append(videoArgs, ffmpeg.HWEncoderArgs(mode.codec, transcodeCRF)...)

	opts := transcoder.TranscodeOptions{
		OutputPath: tmpFn,
		VideoCodec: mode.codec,
		VideoArgs:  videoArgs,

		ExtraInputArgs:  append(append([]string{}, inputArgs...), g.FFMpegConfig.GetTranscodeInputArgs()...),
		ExtraOutputArgs: g.FFMpegConfig.GetTranscodeOutputArgs(),
	}
	if withAudio {
		opts.AudioCodec = ffmpeg.AudioCodecAAC
	} else {
		var audioArgs ffmpeg.Args
		opts.AudioArgs = audioArgs.SkipAudio()
	}

	return transcoder.Transcode(input, opts)
}

// tryHWTranscode attempts the transcode on the GPU. It returns true when the
// output was written. If the GPU cannot do it, it returns false with a nil
// error and the caller falls back to software encoding. The error is only set
// when the job was cancelled, in which case there must be no fallback.
func (g Generator) tryHWTranscode(lockCtx *fsutil.LockContext, input, tmpFn string, options TranscodeOptions, withAudio bool) (bool, error) {
	for _, mode := range g.hwTranscodeModes(lockCtx, options) {
		args := g.hwTranscodeArgs(input, tmpFn, options, withAudio, mode)
		err := g.generate(lockCtx, args)
		if err == nil {
			logger.Debugf("[transcode] encoded %s with %s", input, mode.name)
			return true, nil
		}
		if ctxErr := lockCtx.Err(); ctxErr != nil {
			return false, ctxErr
		}
		logger.Warnf("[transcode] hardware encoding (%s) failed for %s, trying another way: %v", mode.name, input, err)
		_ = os.Remove(tmpFn)
	}
	return false, nil
}
