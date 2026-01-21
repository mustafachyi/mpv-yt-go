//go:build linux

package ui

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var (
	clipboardBin  string
	clipboardArgs []string
	clipboardOnce sync.Once
)

func resolveClipboard() {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if _, err := exec.LookPath("wl-paste"); err == nil {
			clipboardBin = "wl-paste"
			return
		}
	}

	if _, err := exec.LookPath("xclip"); err == nil {
		clipboardBin = "xclip"
		clipboardArgs = []string{"-o", "-selection", "clipboard"}
		return
	}

	if _, err := exec.LookPath("xsel"); err == nil {
		clipboardBin = "xsel"
		clipboardArgs = []string{"-ob"}
		return
	}
}

func getClipboard() string {
	clipboardOnce.Do(resolveClipboard)
	if clipboardBin == "" {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, clipboardBin, clipboardArgs...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
