//go:build linux

package ui

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

func getClipboard() string {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	var bin string
	var args []string

	if os.Getenv("WAYLAND_DISPLAY") != "" {
		bin = "wl-paste"
	} else {
		bin = "xclip"
		args = []string{"-o", "-selection", "clipboard"}
	}

	if _, err := exec.LookPath(bin); err != nil {
		if bin == "xclip" {
			bin = "xsel"
			args = []string{"-ob"}
			if _, err := exec.LookPath(bin); err != nil {
				return ""
			}
		} else {
			return ""
		}
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
