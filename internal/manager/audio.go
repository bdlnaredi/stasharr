package manager

// Stasharr audio support: audio files are imported as scenes. They have no
// picture, so the cover is the embedded album art when there is one, or a
// drawing of the sound wave, and the video-only generators are skipped.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
)

// AudioMimeType returns the browser mime type of an audio file.
func AudioMimeType(path string) string {
	switch strings.ToLower(strings.TrimPrefix(filepath.Ext(path), ".")) {
	case "mp3":
		return "audio/mpeg"
	case "m4a", "m4b", "mp4":
		return "audio/mp4"
	case "aac":
		return "audio/aac"
	case "flac":
		return "audio/flac"
	case "wav":
		return "audio/wav"
	case "ogg", "oga":
		return "audio/ogg"
	case "opus":
		return "audio/ogg; codecs=opus"
	case "webm":
		return "audio/webm"
	default:
		return "audio/mpeg"
	}
}

// browserAudioCodecs are audio codecs browsers play without converting.
var browserAudioCodecs = map[string]bool{
	"mp3": true, "aac": true, "flac": true, "opus": true, "vorbis": true,
	"pcm_s16le": true, "pcm_s24le": true, "pcm_u8": true, "alac": false,
}

// AudioPlaysInBrowser reports whether the file can be streamed as it is.
func AudioPlaysInBrowser(f *models.VideoFile) bool {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(f.Path), "."))
	if ext == "wma" {
		return false
	}
	return browserAudioCodecs[strings.ToLower(f.AudioCodec)]
}

// audioCover makes a cover image for an audio file: the embedded album art
// if there is one, otherwise a picture of the sound wave.
func audioCover(ctx context.Context, path string) ([]byte, error) {
	ff := GetInstance().FFMpeg
	if ff == nil {
		return nil, errors.New("ffmpeg not available")
	}

	art := []string{"-hide_banner", "-nostats", "-v", "error", "-i", path,
		"-map", "0:v:0", "-frames:v", "1", "-vf", "scale='min(1280,iw)':-2",
		"-c:v", "mjpeg", "-q:v", "3", "-f", "image2", "pipe:1"}
	if out, err := runFFmpegCapture(ctx, art); err == nil && len(out) > 0 {
		return out, nil
	}

	wave := []string{"-hide_banner", "-nostats", "-v", "error", "-i", path,
		"-filter_complex", "[0:a:0]aformat=channel_layouts=mono,showwavespic=s=1280x720:colors=0x48aff0:scale=sqrt[w];color=c=0x202b33:s=1280x720[bg];[bg][w]overlay=format=auto",
		"-frames:v", "1", "-c:v", "mjpeg", "-q:v", "3", "-f", "image2", "pipe:1"}
	out, err := runFFmpegCapture(ctx, wave)
	if err != nil {
		return nil, fmt.Errorf("drawing sound wave: %w", err)
	}
	if len(out) == 0 {
		return nil, errors.New("no cover made")
	}
	return out, nil
}

// StreamAudioMp3 converts an audio file to mp3 while it plays, for formats
// browsers cannot play, starting at the "start" query value in seconds.
func StreamAudioMp3(w http.ResponseWriter, r *http.Request, f *models.VideoFile) {
	ff := GetInstance().FFMpeg
	if ff == nil {
		http.Error(w, "ffmpeg not available", http.StatusServiceUnavailable)
		return
	}
	start, _ := strconv.ParseFloat(r.URL.Query().Get("start"), 64)

	args := []string{"-hide_banner", "-nostats", "-v", "error"}
	if start > 0 {
		args = append(args, "-ss", strconv.FormatFloat(start, 'f', 2, 64))
	}
	args = append(args, "-i", f.Path, "-vn", "-sn", "-dn",
		"-c:a", "libmp3lame", "-b:a", "192k", "-f", "mp3", "pipe:1")

	cmd := ff.Command(r.Context(), args)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := cmd.Start(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "no-cache")
	if _, err := io.Copy(w, stdout); err != nil {
		logger.Debugf("[audio] stream ended: %v", err)
	}
	_ = cmd.Wait()
}
