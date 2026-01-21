package mpv

import (
	"fmt"
	"mpy-yt/internal/models"
	"mpy-yt/internal/proxy"
	"os/exec"
)

func IsAvailable() bool {
	_, err := exec.LookPath("mpv")
	return err == nil
}

func Launch(title, thumbUrl string, video *models.VideoStream, audio models.AudioStream) error {
	var vStream, aStream *models.Stream
	if video != nil {
		vStream = &video.Stream
	}
	aStream = &audio.Stream

	srv, vUrl, aUrl, err := proxy.Start(vStream, aStream)
	if err != nil {
		return fmt.Errorf("failed to start proxy: %w", err)
	}
	defer srv.Close()

	var args []string

	if video != nil {
		args = []string{
			"--title=" + title,
			"--force-media-title= ",
			"--keep-open=yes",
			"--cache=yes",
			"--demuxer-max-bytes=256MiB",
			vUrl,
			"--audio-file=" + aUrl,
		}
	} else if thumbUrl != "" {
		args = []string{
			"--title=" + title,
			"--force-media-title= ",
			"--keep-open=yes",
			"--cache=yes",
			"--demuxer-max-bytes=256MiB",
			aUrl,
			"--external-file=" + thumbUrl,
			"--vid=1",
			"--image-display-duration=inf",
			"--force-window=immediate",
			"--video-unscaled=yes",
			"--terminal=no",
		}
	} else {
		args = []string{
			"--title=" + title,
			"--force-media-title= ",
			"--keep-open=yes",
			"--cache=yes",
			"--demuxer-max-bytes=256MiB",
			aUrl,
			"--force-window",
		}
	}

	cmd := exec.Command("mpv", args...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error launching mpv: %w", err)
	}

	return cmd.Wait()
}
