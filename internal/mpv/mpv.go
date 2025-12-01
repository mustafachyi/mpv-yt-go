package mpv

import (
	"fmt"
	"mpy-yt/internal/models"
	"os/exec"
)

func IsAvailable() bool {
	_, err := exec.LookPath("mpv")
	return err == nil
}

func Launch(title, thumbUrl string, video *models.VideoStream, audio models.AudioStream) error {
	fmt.Print("\033[H\033[2J")

	args := []string{
		"--title=" + title,
		"--force-media-title= ",
		"--keep-open=yes",
	}

	if video != nil {
		args = append(args, video.Url, "--audio-file="+audio.Url)
		fmt.Printf("\nPlaying: %s [%s / %s]\n", title, video.Quality, audio.Name)
	} else {
		args = append(args, audio.Url)
		if thumbUrl != "" {
			args = append(args,
				"--external-file="+thumbUrl,
				"--vid=1",
				"--image-display-duration=inf",
				"--force-window=immediate",
				"--video-unscaled=yes",
				"--terminal=no",
			)
		} else {
			args = append(args, "--force-window")
		}
		fmt.Printf("\nPlaying: %s [Audio only / %s]\n", title, audio.Name)
	}

	cmd := exec.Command("mpv", args...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error launching mpv: %w", err)
	}
	return nil
}
