package main

import (
	"flag"
	"fmt"
	"mpy-yt/internal/models"
	"mpy-yt/internal/mpv"
	"mpy-yt/internal/ui"
	"mpy-yt/internal/youtube"
	"os"
)

func main() {
	qualityFlag := flag.String("q", "", "Stream quality")
	qualityLong := flag.String("quality", "", "Stream quality")
	langFlag := flag.String("l", "", "Audio language")
	langLong := flag.String("language", "", "Audio language")
	audioFlag := flag.Bool("a", false, "Play audio only")
	audioLong := flag.Bool("audio", false, "Play audio only")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <identifier>\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	quality := *qualityFlag
	if quality == "" {
		quality = *qualityLong
	}
	lang := *langFlag
	if lang == "" {
		lang = *langLong
	}
	audioOnly := *audioFlag || *audioLong

	args := flag.Args()
	var identifier string
	if len(args) > 0 {
		identifier = args[0]
		if youtube.ExtractVideoId(identifier) == "" {
			fmt.Fprintf(os.Stderr, "Error: Invalid YouTube URL or Video ID: '%s'\n", identifier)
			os.Exit(1)
		}
	}

	if !mpv.IsAvailable() {
		fmt.Fprintln(os.Stderr, "Error: 'mpv' executable not found in your system's PATH.")
		os.Exit(1)
	}

	id := identifier
	if id == "" {
		id = ui.GetIdentifierFromInput()
	}
	if id == "" {
		fmt.Fprintln(os.Stderr, "Error: No identifier provided.")
		os.Exit(1)
	}

	videoId := youtube.ExtractVideoId(id)
	if videoId == "" {
		fmt.Fprintf(os.Stderr, "Error: Invalid YouTube URL or Video ID: '%s'\n", id)
		os.Exit(1)
	}

	playerData, err := youtube.GetPlayerData(videoId)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	selection := ui.GetStreamSelection(playerData, quality, lang, audioOnly)
	if selection == nil {
		os.Exit(0)
	}

	var mpvErr error
	switch s := selection.(type) {
	case models.VideoSelection:
		mpvErr = mpv.Launch(playerData.Title, playerData.ThumbnailUrl, &s.Video, s.Audio)
	case models.AudioSelection:
		mpvErr = mpv.Launch(playerData.Title, playerData.ThumbnailUrl, nil, s.Audio)
	}

	if mpvErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", mpvErr)
		os.Exit(1)
	}
	fmt.Print("\033[H\033[2J")
}
