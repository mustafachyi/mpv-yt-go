package mpv

import (
	"fmt"
	"mpy-yt/internal/models"
	"mpy-yt/internal/proxy"
	"os/exec"
)

func Launch(title, thumbUrl string, video *models.VideoStream, audio *models.AudioStream) error {
	var vStream, aStream *models.Stream
	if video != nil {
		vStream = &video.Stream
	}
	if audio != nil {
		aStream = &audio.Stream
	}

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
			"--no-ytdl",
			"--hwdec=auto",
			"--profile=fast",
			"--gpu-dumb-mode=yes",
			"--vd-lavc-skiploopfilter=nonref",
			"--vd-lavc-threads=0",
			"--sws-allow-zimg=no",
			"--sws-fast",
			"--framedrop=vo",
			"--priority=high",
			"--force-window=yes",
			"--terminal=no",
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
			"--no-ytdl",
			"--profile=fast",
			"--sws-fast",
			"--framedrop=vo",
			"--force-window=yes",
			"--terminal=no",
			aUrl,
			"--external-file=" + thumbUrl,
			"--vid=1",
			"--image-display-duration=inf",
			"--video-unscaled=yes",
		}
	} else {
		args = []string{
			"--title=" + title,
			"--force-media-title= ",
			"--keep-open=yes",
			"--cache=yes",
			"--demuxer-max-bytes=256MiB",
			"--no-ytdl",
			"--terminal=no",
			"--vo=null",
			"--vid=no",
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
