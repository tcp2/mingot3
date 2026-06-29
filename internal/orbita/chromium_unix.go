//go:build !windows

// chromium_unix.go — Spawn chromium và quản lý process lifecycle (Non-Windows).
package orbita

import (
	"os/exec"
	"syscall"
)

// spawn khởi chạy browser với args, detached khỏi parent process.
// Trả về *exec.Cmd đã Start() (chưa Wait).
func spawn(browserExe string, args []string) (*exec.Cmd, error) {
	cmd := exec.Command(browserExe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	return cmd, cmd.Start()
}
