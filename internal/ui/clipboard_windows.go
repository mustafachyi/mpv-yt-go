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

	n := int(sz / 2)

	var s []uint16
	sh := (*struct {
		Data uintptr
		Len  int
		Cap  int
	})(unsafe.Pointer(&s))
	sh.Data = p
	sh.Len = n
	sh.Cap = n

	end := 0
	for end < n && s[end] != 0 {
		end++
	}

	return syscall.UTF16ToString(s[:end])
}
