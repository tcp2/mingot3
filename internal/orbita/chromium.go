//go:build windows

// chromium.go — Spawn chrome.exe và quản lý process lifecycle (Windows only).
package orbita

import (
	"os/exec"
	"syscall"
)

// spawn khởi chạy chrome.exe với args, detached khỏi parent process.
// Trả về *exec.Cmd đã Start() (chưa Wait).
func spawn(browserExe string, args []string) (*exec.Cmd, error) {
	cmd := exec.Command(browserExe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	return cmd, cmd.Start()
}
