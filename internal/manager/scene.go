package manager

import (
	"fmt"
	"net/url"

	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/ffmpeg"
	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/models"
)

type SceneStreamEndpoint struct {
	URL      string  `json:"url"`
	MimeType *string `json:"mime_type"`
	Label    *string `json:"label"`
}

type endpointType struct {
	label     string
	mimeType  string
	extension string
}

var (
	directEndpointType = endpointType{
		label:     "Direct stream",
		mimeType:  ffmpeg.MimeMp4Video,
		extension: "",
	}
	mp4EndpointType = endpointType{
		label:     "MP4",
		mimeType:  ffmpeg.MimeMp4Video,
		extension: ".mp4",
	}
	mkvEndpointType = endpointType{
		label: "MKV",
		// use mp4 mimetype to trick the client, since many clients won't try mkv
		mimeType:  ffmpeg.MimeMp4Video,
		extension: ".mkv",
	}
	webmEndpointType = endpointType{
		label:     "WEBM",
		mimeType:  ffmpeg.MimeWebmVideo,
		extension: ".webm",
	}
	hlsEndpointType = endpointType{
		label:     "HLS",
		mimeType:  ffmpeg.MimeHLS,
		extension: ".m3u8",
	}
	// Stasharr: adaptive HLS, see AdaptiveVariants
	hlsAutoEndpointType = endpointType{
		label:     "HLS Auto",
		mimeType:  ffmpeg.MimeHLS,
		extension: ".auto.m3u8",
	}
	dashEndpointType = endpointType{
		label:     "DASH",
		mimeType:  ffmpeg.MimeDASH,
		extension: ".mpd",
	}
)

func GetVideoFileContainer(file *models.VideoFile) (ffmpeg.Container, error) {
	var container ffmpeg.Container
	format := file.Format
	if format != "" {
		container = ffmpeg.Container(format)
	} else { // container isn't in the DB
		// shouldn't happen, fallback to ffprobe
		ffprobe := GetInstance().FFProbe
		tmpVideoFile, err := ffprobe.NewVideoFile(file.Path)
		if err != nil {
			return ffmpeg.Container(""), fmt.Errorf("error reading video file: %v", err)
		}

		return ffmpeg.MatchContainer(tmpVideoFile.Container, file.Path)
	}

	return container, nil
}

func GetSceneStreamPaths(scene *models.Scene, directStreamURL *url.URL, maxStreamingTranscodeSize models.StreamingResolutionEnum) ([]*SceneStreamEndpoint, error) {
	if scene == nil {
		return nil, fmt.Errorf("nil scene")
	}

	pf := scene.Files.Primary()
	if pf == nil {
		return nil, nil
	}

	// convert StreamingResolutionEnum to ResolutionEnum
	maxStreamingResolution := models.ResolutionEnum(maxStreamingTranscodeSize)
	sceneResolution := models.GetMinResolution(pf)
	includeSceneStreamPath := func(streamingResolution models.StreamingResolutionEnum) bool {
		var minResolution int
		if streamingResolution == models.StreamingResolutionEnumOriginal {
			minResolution = sceneResolution
		} else {
			// convert StreamingResolutionEnum to ResolutionEnum so we can get the min
			// resolution
			convertedRes := models.ResolutionEnum(streamingResolution)
			minResolution = convertedRes.GetMinResolution()

			// don't include if scene resolution is smaller than the streamingResolution
			if sceneResolution != 0 && sceneResolution < minResolution {
				return false
			}
		}

		// if we always allow everything, then return true
		if maxStreamingTranscodeSize == models.StreamingResolutionEnumOriginal {
			return true
		}

		return maxStreamingResolution.GetMinResolution() >= minResolution
	}

	makeStreamEndpoint := func(t endpointType, resolution models.StreamingResolutionEnum) *SceneStreamEndpoint {
		url := *directStreamURL
		url.Path += t.extension

		label := t.label

		if resolution != "" {
			v := url.Query()
			v.Set("resolution", resolution.String())
			url.RawQuery = v.Encode()

			switch resolution {
			case models.StreamingResolutionEnumFourK:
				label += " 4K (2160p)"
			case models.StreamingResolutionEnumFullHd:
				label += " Full HD (1080p)"
			case models.StreamingResolutionEnumStandardHd:
				label += " HD (720p)"
			case models.StreamingResolutionEnumStandard:
				label += " Standard (480p)"
			case models.StreamingResolutionEnumLow:
				label += " Low (240p)"
			}
		}

		return &SceneStreamEndpoint{
			URL:      url.String(),
			MimeType: &t.mimeType,
			Label:    &label,
		}
	}

	var endpoints []*SceneStreamEndpoint

	// Stasharr: audio files play as they are, with an mp3 conversion for
	// formats browsers cannot play
	if pf.IsAudioOnly() {
		if AudioPlaysInBrowser(pf) {
			direct := endpointType{label: "Direct stream", mimeType: AudioMimeType(pf.Path)}
			endpoints = append(endpoints, makeStreamEndpoint(direct, ""))
		}
		mp3 := endpointType{label: "MP3", mimeType: "audio/mpeg", extension: ".mp3"}
		endpoints = append(endpoints, makeStreamEndpoint(mp3, ""))
		return endpoints, nil
	}

	// direct stream should only apply when the audio codec is supported
	audioCodec := ffmpeg.MissingUnsupported
	if pf.AudioCodec != "" {
		audioCodec = ffmpeg.ProbeAudioCodec(pf.AudioCodec)
	}

	// don't care if we can't get the container
	container, _ := GetVideoFileContainer(pf)

	if HasTranscode(scene, config.GetInstance().GetVideoFileNamingAlgorithm()) || ffmpeg.IsValidAudioForContainer(audioCodec, container) {
		endpoints = append(endpoints, makeStreamEndpoint(directEndpointType, ""))
	}

	// only add mkv stream endpoint if the scene container is an mkv already
	if container == ffmpeg.Matroska {
		endpoints = append(endpoints, makeStreamEndpoint(mkvEndpointType, ""))
	}

	mp4Streams := []*SceneStreamEndpoint{}
	webmStreams := []*SceneStreamEndpoint{}
	hlsStreams := []*SceneStreamEndpoint{}
	dashStreams := []*SceneStreamEndpoint{}

	if includeSceneStreamPath(models.StreamingResolutionEnumOriginal) {
		mp4Streams = append(mp4Streams, makeStreamEndpoint(mp4EndpointType, models.StreamingResolutionEnumOriginal))
		webmStreams = append(webmStreams, makeStreamEndpoint(webmEndpointType, models.StreamingResolutionEnumOriginal))
		hlsStreams = append(hlsStreams, makeStreamEndpoint(hlsEndpointType, models.StreamingResolutionEnumOriginal))
		dashStreams = append(dashStreams, makeStreamEndpoint(dashEndpointType, models.StreamingResolutionEnumOriginal))
	}

	if includeSceneStreamPath(models.StreamingResolutionEnumFourK) {
		mp4Streams = append(mp4Streams, makeStreamEndpoint(mp4EndpointType, models.StreamingResolutionEnumFourK))
		webmStreams = append(webmStreams, makeStreamEndpoint(webmEndpointType, models.StreamingResolutionEnumFourK))
		hlsStreams = append(hlsStreams, makeStreamEndpoint(hlsEndpointType, models.StreamingResolutionEnumFourK))
		dashStreams = append(dashStreams, makeStreamEndpoint(dashEndpointType, models.StreamingResolutionEnumFourK))
	}

	if includeSceneStreamPath(models.StreamingResolutionEnumFullHd) {
		mp4Streams = append(mp4Streams, makeStreamEndpoint(mp4EndpointType, models.StreamingResolutionEnumFullHd))
		webmStreams = append(webmStreams, makeStreamEndpoint(webmEndpointType, models.StreamingResolutionEnumFullHd))
		hlsStreams = append(hlsStreams, makeStreamEndpoint(hlsEndpointType, models.StreamingResolutionEnumFullHd))
		dashStreams = append(dashStreams, makeStreamEndpoint(dashEndpointType, models.StreamingResolutionEnumFullHd))
	}

	if includeSceneStreamPath(models.StreamingResolutionEnumStandardHd) {
		mp4Streams = append(mp4Streams, makeStreamEndpoint(mp4EndpointType, models.StreamingResolutionEnumStandardHd))
		webmStreams = append(webmStreams, makeStreamEndpoint(webmEndpointType, models.StreamingResolutionEnumStandardHd))
		hlsStreams = append(hlsStreams, makeStreamEndpoint(hlsEndpointType, models.StreamingResolutionEnumStandardHd))
		dashStreams = append(dashStreams, makeStreamEndpoint(dashEndpointType, models.StreamingResolutionEnumStandardHd))
	}

	if includeSceneStreamPath(models.StreamingResolutionEnumStandard) {
		mp4Streams = append(mp4Streams, makeStreamEndpoint(mp4EndpointType, models.StreamingResolutionEnumStandard))
		webmStreams = append(webmStreams, makeStreamEndpoint(webmEndpointType, models.StreamingResolutionEnumStandard))
		hlsStreams = append(hlsStreams, makeStreamEndpoint(hlsEndpointType, models.StreamingResolutionEnumStandard))
		dashStreams = append(dashStreams, makeStreamEndpoint(dashEndpointType, models.StreamingResolutionEnumStandard))
	}

	if includeSceneStreamPath(models.StreamingResolutionEnumLow) {
		mp4Streams = append(mp4Streams, makeStreamEndpoint(mp4EndpointType, models.StreamingResolutionEnumLow))
		webmStreams = append(webmStreams, makeStreamEndpoint(webmEndpointType, models.StreamingResolutionEnumLow))
		hlsStreams = append(hlsStreams, makeStreamEndpoint(hlsEndpointType, models.StreamingResolutionEnumLow))
		dashStreams = append(dashStreams, makeStreamEndpoint(dashEndpointType, models.StreamingResolutionEnumLow))
	}

	// Stasharr: one HLS stream with every size, so players switch to a
	// smaller one when the connection slows down
	if len(hlsStreams) >= 2 {
		hlsStreams = append([]*SceneStreamEndpoint{makeStreamEndpoint(hlsAutoEndpointType, "")}, hlsStreams...)
	}

	endpoints = append(endpoints, mp4Streams...)
	endpoints = append(endpoints, webmStreams...)
	endpoints = append(endpoints, hlsStreams...)
	endpoints = append(endpoints, dashStreams...)

	return endpoints, nil
}

// HasTranscode returns true if a transcoded video exists for the provided
// scene. It will check using the OSHash of the scene first, then fall back
// to the checksum.
func HasTranscode(scene *models.Scene, fileNamingAlgo models.HashAlgorithm) bool {
	if scene == nil {
		return false
	}

	sceneHash := scene.GetHash(fileNamingAlgo)
	if sceneHash == "" {
		return false
	}

	transcodePath := instance.Paths.Scene.GetTranscodePath(sceneHash)
	ret, _ := fsutil.FileExists(transcodePath)
	return ret
}

// HLSVariant is one size in the adaptive HLS stream.
type HLSVariant struct {
	Resolution models.StreamingResolutionEnum
	Width      int
	Height     int
	Bandwidth  int
}

// AdaptiveVariants lists the sizes for the adaptive HLS stream of a video,
// smallest first: every standard size up to the video's own and the
// largest the server is set to convert to.
func AdaptiveVariants(vf *models.VideoFile, maxTranscode models.StreamingResolutionEnum) []HLSVariant {
	if vf == nil || vf.Height == 0 || vf.Width == 0 {
		return nil
	}
	short := min(vf.Width, vf.Height)
	sizes := []struct {
		res       models.StreamingResolutionEnum
		height    int
		bandwidth int
	}{
		{models.StreamingResolutionEnumLow, 240, 700_000},
		{models.StreamingResolutionEnumStandard, 480, 1_600_000},
		{models.StreamingResolutionEnumStandardHd, 720, 3_200_000},
		{models.StreamingResolutionEnumFullHd, 1080, 6_000_000},
		{models.StreamingResolutionEnumFourK, 2160, 18_000_000},
	}
	limit := models.ResolutionEnum(maxTranscode).GetMinResolution()
	if maxTranscode == models.StreamingResolutionEnumOriginal || maxTranscode == "" {
		limit = 1 << 30
	}
	scale := func(h int) (int, int) {
		// the short side gets the size; keep the shape, even numbers
		if vf.Width >= vf.Height {
			w := int(float64(h)*float64(vf.Width)/float64(vf.Height)/2) * 2
			return w, h
		}
		w := int(float64(h)*float64(vf.Height)/float64(vf.Width)/2) * 2
		return h, w
	}
	var ret []HLSVariant
	for _, s := range sizes {
		if s.height > short+short/10 || s.height > limit {
			continue
		}
		w, h := scale(s.height)
		ret = append(ret, HLSVariant{Resolution: s.res, Width: w, Height: h, Bandwidth: s.bandwidth})
	}
	// a video between the standard sizes also gets its own size on top
	if len(ret) > 0 && short > scaleShort(ret[len(ret)-1])*11/10 && short <= limit {
		bw := int(vf.BitRate)
		if bw <= 0 {
			bw = ret[len(ret)-1].Bandwidth * 2
		}
		ret = append(ret, HLSVariant{Resolution: models.StreamingResolutionEnumOriginal, Width: vf.Width, Height: vf.Height, Bandwidth: bw})
	}
	return ret
}

func scaleShort(v HLSVariant) int { return min(v.Width, v.Height) }
