package models

// IsAudioOnly reports whether the file has sound but no picture, such as an
// mp3. Cover art embedded in audio files is not counted as a picture.
func (f VideoFile) IsAudioOnly() bool {
	return f.VideoCodec == "" && f.Width == 0 && f.Height == 0 && f.AudioCodec != ""
}
