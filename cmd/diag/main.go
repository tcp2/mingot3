// cmd/diag/main.go — Diagnostic tool kiểm tra system state mà không launch browser.
//
// Chạy: go run ./cmd/diag
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mingot/internal/storage"
	"mingot/internal/util"
)

func main() {
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("  Mingot Diagnostic Tool")
	fmt.Println("═══════════════════════════════════════")

	checkPaths()
	checkTokens()
	checkProfiles()
}

// ─── Paths ────────────────────────────────────────────────────────────────────

func checkPaths() {
	fmt.Println("\n[1] Kiểm tra đường dẫn hệ thống")

	paths := []struct {
		label string
		path  string
	}{
		{"Browser exe  ", storage.BrowserExe()},
		{"Zero profile ", storage.ZeroFile()},
		{"Profiles dir ", storage.GologinDir()},
	}

	for _, p := range paths {
		_, err := os.Stat(p.path)
		status := "✓"
		if err != nil {
			status = "✗ KHÔNG TỒN TẠI"
		}
		fmt.Printf("  %s %s  →  %s\n", p.label, status, p.path)
	}
}

// ─── Tokens ───────────────────────────────────────────────────────────────────

func checkTokens() {
	fmt.Println("\n[2] Kiểm tra GoLogin tokens")

	dirs := storage.TokenStoreDirs()
	for _, d := range dirs {
		if _, err := os.Stat(d); err == nil {
			fmt.Printf("  LevelDB dir tìm thấy: %s\n", d)
		}
	}

	tokens := util.ExtractGoLoginTokens(dirs)
	if len(tokens) == 0 {
		fmt.Println("  ✗ Không tìm thấy token. GoLogin chưa login?")
		return
	}

	fmt.Printf("  ✓ Tìm thấy %d token(s):\n", len(tokens))
	for i, t := range tokens {
		preview := t.Raw
		if len(preview) > 40 {
			preview = preview[:20] + "..." + preview[len(preview)-10:]
		}
		fmt.Printf("    [%d] userID=%s  token=%s\n", i+1, t.UserID, preview)
	}
}

// ─── Profiles ─────────────────────────────────────────────────────────────────

func checkProfiles() {
	fmt.Println("\n[3] Kiểm tra profiles hiện có")

	gologinDir := storage.GologinDir()
	entries, err := os.ReadDir(gologinDir)
	if err != nil {
		fmt.Printf("  ✗ Không đọc được thư mục profiles: %v\n", err)
		return
	}

	var profiles []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "MG-") {
			profiles = append(profiles, e.Name())
		}
	}

	if len(profiles) == 0 {
		fmt.Println("  (Chưa có profile nào)")
		return
	}

	fmt.Printf("  ✓ Tìm thấy %d profile(s):\n", len(profiles))
	for _, id := range profiles {
		info := readProfileInfo(filepath.Join(gologinDir, id))
		running := checkRunning(filepath.Join(gologinDir, id))
		status := "stopped"
		if running {
			status = "🟢 RUNNING"
		}
		fmt.Printf("    %-25s  name=%-20s  status=%s\n", id, info.Name, status)
		if running && info.Port != 0 {
			fmt.Printf("    └─ debug port: %d  (http://127.0.0.1:%d)\n", info.Port, info.Port)
		}
	}
}

type profileInfo struct {
	Name string
	Port int
}

func readProfileInfo(profilePath string) profileInfo {
	prefsPath := filepath.Join(profilePath, "Default", "Preferences")
	data, err := os.ReadFile(prefsPath)
	if err != nil {
		return profileInfo{Name: "(no prefs)"}
	}
	var prefs struct {
		Gologin struct {
			Name string `json:"name"`
			Port *int   `json:"port"`
		} `json:"gologin"`
	}
	if err := json.Unmarshal(data, &prefs); err != nil {
		return profileInfo{Name: "(parse error)"}
	}
	info := profileInfo{Name: prefs.Gologin.Name}
	if prefs.Gologin.Port != nil {
		info.Port = *prefs.Gologin.Port
	}
	return info
}

func checkRunning(profilePath string) bool {
	if _, err := os.Stat(profilePath); err != nil {
		return false
	}
	return os.Rename(profilePath, profilePath) != nil
}
