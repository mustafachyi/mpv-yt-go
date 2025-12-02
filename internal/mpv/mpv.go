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
	var args []string

	if video != nil {
		args = make([]string, 0, 5)
		args = append(args,
			"--title="+title,
			"--force-media-title= ",
			"--keep-open=yes",
			video.Url,
			"--audio-file="+audio.Url,
		)
	} else if thumbUrl != "" {
		args = make([]string, 0, 10)
		args = append(args,
			"--title="+title,
			"--force-media-title= ",
			"--keep-open=yes",
			audio.Url,
			"--external-file="+thumbUrl,
			"--vid=1",
			"--image-display-duration=inf",
			"--force-window=immediate",
			"--video-unscaled=yes",
			"--terminal=no",
		)
	} else {
		args = make([]string, 0, 5)
		args = append(args,
			"--title="+title,
			"--force-media-title= ",
			"--keep-open=yes",
			audio.Url,
			"--force-window",
		)
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
