//go:build windows

package ui

import (
	"syscall"
	"unsafe"
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	openClipboard    = user32.NewProc("OpenClipboard")
	closeClipboard   = user32.NewProc("CloseClipboard")
	getClipboardData = user32.NewProc("GetClipboardData")
	globalLock       = kernel32.NewProc("GlobalLock")
	globalUnlock     = kernel32.NewProc("GlobalUnlock")
	globalSize       = kernel32.NewProc("GlobalSize")
)

const cfUnicodeText = 13

func getClipboard() string {
	r, _, _ := openClipboard.Call(0)
	if r == 0 {
		return ""
	}
	defer closeClipboard.Call()

	h, _, _ := getClipboardData.Call(cfUnicodeText)
	if h == 0 {
		return ""
	}

	p, _, _ := globalLock.Call(h)
	if p == 0 {
		return ""
	}
	defer globalUnlock.Call(h)

	sz, _, _ := globalSize.Call(h)
	if sz == 0 {
		return ""
	}

	var s []uint16
	hdr := (*struct {
		Data uintptr
		Len  int
		Cap  int
	})(unsafe.Pointer(&s))

	hdr.Data = p
	hdr.Len = int(sz / 2)
	hdr.Cap = int(sz / 2)

	n := 0
	for n < len(s) && s[n] != 0 {
		n++
	}

	return syscall.UTF16ToString(s[:n])
}
