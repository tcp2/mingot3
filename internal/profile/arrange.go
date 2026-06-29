// arrange.go — Sắp xếp tất cả cửa sổ browser đang chạy theo grid.
package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"mingot/internal/cdp"
)

// ArrangeResult chứa số cửa sổ đã được sắp xếp.
type ArrangeResult struct {
	ArrangedCount int `json:"arrangedCount"`
}

// ArrangeWindows sắp xếp tất cả browser đang chạy theo grid layout.
func (m *Manager) ArrangeWindows(screenW, screenH int) (*ArrangeResult, error) {
	running := collectRunning()
	sort.Slice(running, func(i, j int) bool {
		return running[i].id < running[j].id
	})

	count := 0
	for i, p := range running {
		cfg := m.grid.Config(i, screenW, screenH)
		if err := cdp.SetWindowBounds(p.port, cfg.X, cfg.Y, cfg.W, cfg.H); err != nil {
			fmt.Printf("[ARRANGE] port %d: %v\n", p.port, err)
			continue
		}
		count++
	}
	return &ArrangeResult{ArrangedCount: count}, nil
}

type runningProfile struct {
	id   string
	port int
}

func collectRunning() []runningProfile {
	entries, err := os.ReadDir(gologinDir())
	if err != nil {
		return nil
	}

	var result []runningProfile
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		profilePath := filepath.Join(gologinDir(), id)
		if !isRunning(profilePath) {
			continue
		}
		port, err := readPort(id)
		if err != nil || port == 0 {
			continue
		}
		result = append(result, runningProfile{id: id, port: port})
	}
	return result
}

func readPort(id string) (int, error) {
	prefsPath := filepath.Join(gologinDir(), id, "Default", "Preferences")
	data, err := os.ReadFile(prefsPath)
	if err != nil {
		return 0, err
	}
	var prefs struct {
		Gologin struct {
			Port *int `json:"port"`
		} `json:"gologin"`
	}
	if err := json.Unmarshal(data, &prefs); err != nil {
		return 0, err
	}
	if prefs.Gologin.Port == nil {
		return 0, nil
	}
	return *prefs.Gologin.Port, nil
}
