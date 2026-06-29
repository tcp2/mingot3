package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"mingot/internal/cdp"
	"mingot/internal/orbita"
)

// ─── List ─────────────────────────────────────────────────────────────────────

// List trả về danh sách tất cả profiles, sắp xếp mới nhất trước.
func (m *Manager) List() ([]Summary, error) {
	dir := gologinDir()
	if _, err := os.Stat(dir); err != nil {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		return []Summary{}, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var profiles []Summary
	for _, e := range entries {
		if !e.IsDir() || len(e.Name()) < 3 || e.Name()[:3] != "MG-" {
			continue
		}
		s := readSummary(e.Name())
		if !isRunning(filepath.Join(gologinDir(), e.Name())) {
			m.grid.Free(e.Name()) // auto cleanup slot nếu browser đã tắt
			s.Status = "stopped"
			s.Port = nil
		}
		profiles = append(profiles, s)
	}

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].ID > profiles[j].ID // mới nhất trước
	})
	return profiles, nil
}

func readSummary(id string) Summary {
	s := Summary{ID: id, Name: id, Status: "stopped", Folder: "Uncategorized"}
	prefsPath := filepath.Join(gologinDir(), id, "Default", "Preferences")

	data, err := os.ReadFile(prefsPath)
	if err != nil {
		return s
	}
	var prefs struct {
		Gologin struct {
			Name   string `json:"name"`
			Port   *int   `json:"port"`
			Folder string `json:"folder"`
		} `json:"gologin"`
	}
	if err := json.Unmarshal(data, &prefs); err != nil {
		return s
	}

	if prefs.Gologin.Name != "" {
		s.Name = prefs.Gologin.Name
	}
	s.Port = prefs.Gologin.Port
	if prefs.Gologin.Folder != "" {
		s.Folder = prefs.Gologin.Folder
	}
	if isRunning(filepath.Join(gologinDir(), id)) {
		s.Status = "running"
	}
	return s
}

// isRunning kiểm tra profile có đang chạy bằng cách thử rename thư mục sang chính nó.
// Trên Windows, thư mục bị lock khi Chromium đang chạy sẽ trả EBUSY/EPERM.
func isRunning(profilePath string) bool {
	if _, err := os.Stat(profilePath); err != nil {
		return false
	}
	return os.Rename(profilePath, profilePath) != nil
}

// ─── Create ───────────────────────────────────────────────────────────────────

// Create khởi tạo profile mới, trả về ID và tên.
func (m *Manager) Create() (*CreateResult, error) {
	l, err := orbita.New("")
	if err != nil {
		return nil, err
	}
	if err := l.Init(); err != nil {
		return nil, err
	}
	if err := l.BuildProfile(); err != nil {
		return nil, err
	}
	return &CreateResult{ID: l.ProfileID, Name: l.ProfileName}, nil
}

// ─── Start ────────────────────────────────────────────────────────────────────

// Start khởi chạy profile, trả về debug port.
func (m *Manager) Start(id string, screenW, screenH int) (*StartResult, error) {
	profilePath := filepath.Join(gologinDir(), id)
	if _, err := os.Stat(profilePath); err != nil {
		return nil, notFound("profile không tồn tại")
	}

	slotIndex := m.grid.Allocate(id)
	gridCfg := m.grid.Config(slotIndex, screenW, screenH)

	l, err := orbita.New(id)
	if err != nil {
		m.grid.Free(id)
		return nil, err
	}
	if err := l.Init(); err != nil {
		m.grid.Free(id)
		return nil, err
	}
	if err := l.BuildProfile(); err != nil {
		m.grid.Free(id)
		return nil, err
	}

	onClose := m.makeOnClose(id, profilePath, l.DebugPort)
	if err := l.Launch(&gridCfg, onClose); err != nil {
		m.grid.Free(id)
		return nil, err
	}

	// Vị trí cửa sổ đã được set trước bởi Launch, nhưng đảm bảo thêm lần nữa
	go cdp.SetWindowBounds(l.DebugPort, gridCfg.X, gridCfg.Y, gridCfg.W, gridCfg.H) //nolint:errcheck

	return &StartResult{Port: l.DebugPort}, nil
}

func (m *Manager) makeOnClose(id, profilePath string, _ int) func() {
	return func() {
		fmt.Printf("[PROFILE] Browser %s đã đóng\n", id)
		m.grid.Free(id)
		m.emit("profile:closed", id)

		go func() {
			m.emit("profile:backing-up", id)
			clearPort(profilePath)
			// Tránh import vòng: dùng code sanitize trực tiếp nếu cần hoặc thông qua package.
			// backup.Sanitize được gọi tại đây.
			sanitizeBackupPath(profilePath)
			m.emit("profile:backed-up", id)
		}()
	}
}

func clearPort(profilePath string) {
	prefsPath := filepath.Join(profilePath, "Default", "Preferences")
	prefs, err := readPrefsMap(prefsPath)
	if err != nil {
		return
	}
	gl, ok := prefs["gologin"].(map[string]any)
	if !ok {
		return
	}
	gl["port"] = nil
	_ = writePrefsMap(prefsPath, prefs)
}

// sanitizeBackupPath thực hiện việc sanitize Default/Preferences của profile để chuẩn bị backup
func sanitizeBackupPath(profilePath string) {
	prefsPath := filepath.Join(profilePath, "Default", "Preferences")
	prefs, err := readPrefsMap(prefsPath)
	if err != nil {
		return
	}
	gl, ok := prefs["gologin"].(map[string]any)
	if !ok {
		return
	}
	delete(gl, "port")
	_ = writePrefsMap(prefsPath, prefs)
}

// ─── Stop ─────────────────────────────────────────────────────────────────────

// Stop gửi Browser.close tới profile đang chạy.
func (m *Manager) Stop(id string) error {
	port, err := getProfilePort(id)
	if err != nil {
		return err
	}
	return cdp.CloseBrowser(port)
}

func getProfilePort(id string) (int, error) {
	prefsPath := filepath.Join(gologinDir(), id, "Default", "Preferences")
	data, err := os.ReadFile(prefsPath)
	if err != nil {
		return 0, notFound(fmt.Sprintf("profile %s không tồn tại", id))
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
		return 0, notFound("profile không có port (chưa chạy?)")
	}
	return *prefs.Gologin.Port, nil
}

// ─── Delete ───────────────────────────────────────────────────────────────────

// Delete xoá thư mục profile. Lỗi nếu profile đang chạy hoặc không tồn tại.
func (m *Manager) Delete(id string) error {
	profilePath := filepath.Join(gologinDir(), id)

	if _, err := os.Stat(profilePath); err != nil {
		return notFound("profile không tồn tại")
	}
	if isRunning(profilePath) {
		return profileRunning("không thể xoá profile đang chạy, hãy dừng trước")
	}

	if err := os.RemoveAll(profilePath); err != nil {
		return err
	}

	// Xoá file .zip backup nếu có
	zipPath := filepath.Join(gologinDir(), id+".zip")
	_ = os.Remove(zipPath)

	return nil
}

// ─── Folder ───────────────────────────────────────────────────────────────────

// UpdateFolder gán folder mới cho nhiều profiles cùng lúc.
func (m *Manager) UpdateFolder(ids []string, folder string) error {
	for _, id := range ids {
		if err := setFolder(id, folder); err != nil {
			return fmt.Errorf("profile %s: %w", id, err)
		}
	}
	return nil
}

func setFolder(id, folder string) error {
	prefsPath := filepath.Join(gologinDir(), id, "Default", "Preferences")
	if _, err := os.Stat(prefsPath); err != nil {
		return nil
	}

	prefs, err := readPrefsMap(prefsPath)
	if err != nil {
		return err
	}
	if prefs["gologin"] == nil {
		prefs["gologin"] = map[string]any{}
	}
	prefs["gologin"].(map[string]any)["folder"] = folder
	return writePrefsMap(prefsPath, prefs)
}

// ─── Arrange ──────────────────────────────────────────────────────────────────

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

// ArrangeResult chứa số cửa sổ đã được sắp xếp.
type ArrangeResult struct {
	ArrangedCount int `json:"arrangedCount"`
}
