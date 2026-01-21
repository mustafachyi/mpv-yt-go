package models

type Stream struct {
	Url     string
	Bitrate int64
	Size    int64
}

type VideoStream struct {
	Stream
	Quality string
}

type AudioStream struct {
	Stream
	Language  string
	Name      string
	IsDefault bool
}

type PlayerData struct {
	Title        string
	ThumbnailUrl string
	Videos       []VideoStream
	Audios       []AudioStream
}

type StreamSelection interface {
	isSelection()
}

type VideoSelection struct {
	Video VideoStream
	Audio AudioStream
}

func (VideoSelection) isSelection() {}

type AudioSelection struct {
	Audio AudioStream
}

func (AudioSelection) isSelection() {}
