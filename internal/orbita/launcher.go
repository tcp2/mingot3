// launcher.go — OrbitaLauncher: khởi tạo, build config và launch profile browser.
package orbita

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"mingot/internal/grid"
	"mingot/internal/storage"
	"mingot/internal/util"
)

const tzURL = "https://geo.myip.link"

// Sentinel errors.
var (
	ErrBrowserNotFound = errors.New("orbita browser không tìm thấy")
	ErrProfileLocked   = errors.New("profile đang được mở, vui lòng đóng trước")
	ErrNoToken         = errors.New("không lấy được profile_token (tất cả token bị từ chối)")
)

// Launcher khởi chạy một Orbita browser profile.
type Launcher struct {
	ProfileID   string // ID thư mục (MG-xxx)
	ProfileName string
	DebugPort   int

	profilePath string
	isNew       bool
	auth        []util.Token
	tz          tzInfo
	fingerprint GLProfile
	token       string
	resolution  string
}

// New tạo Launcher cho profile mới (existingID = "").
// New tạo Launcher cho profile hiện có nếu existingID != "".
func New(existingID string) (*Launcher, error) {
	gologinDir := storage.GologinDir()
	l := &Launcher{
		auth:       util.ExtractGoLoginTokens(storage.TokenStoreDirs()),
		resolution: "1920x1080",
	}

	if existingID != "" {
		l.ProfileID = existingID
		l.profilePath = filepath.Join(gologinDir, existingID)
		l.ProfileName = existingID
		l.isNew = false
		l.loadExistingName()
		return l, nil
	}

	if err := os.MkdirAll(gologinDir, 0o755); err != nil {
		return nil, err
	}
	l.isNew = true
	l.ProfileID = "MG-" + util.NewULID()
	l.profilePath = filepath.Join(gologinDir, l.ProfileID)
	l.ProfileName = generateName(gologinDir)
	return l, nil
}

func (l *Launcher) loadExistingName() {
	prefs, err := readPrefs(l.profilePath)
	if err != nil {
		return
	}
	if gl, ok := prefs["gologin"].(map[string]any); ok {
		if name, _ := gl["name"].(string); name != "" {
			l.ProfileName = name
		}
		if sw, sh := intVal(gl, "screenWidth"), intVal(gl, "screenHeight"); sw > 0 && sh > 0 {
			l.resolution = fmt.Sprintf("%dx%d", sw, sh)
		}
	}
}

// Init lấy timezone + fingerprint từ GoLogin API (chỉ cần cho profile mới).
func (l *Launcher) Init() error {
	port, err := util.FreePort()
	if err != nil {
		return fmt.Errorf("lấy port: %w", err)
	}
	l.DebugPort = port

	if !l.isNew {
		return nil
	}

	// Bỏ qua check file zero_profile.json vì đã được embed vào binary.

	tz, err := fetchTZ()
	if err != nil {
		return fmt.Errorf("lấy timezone: %w", err)
	}
	l.tz = tz

	return l.fetchFingerprintWithFallback()
}

func (l *Launcher) fetchFingerprintWithFallback() error {
	stats := l.loadTokenStats()
	sort.Slice(l.auth, func(i, j int) bool {
		return stats[l.auth[i].Raw] < stats[l.auth[j].Raw]
	})

	var lastErr error
	for _, a := range l.auth {
		if err := l.tryFetchFingerprint(a, stats); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr != nil {
		return fmt.Errorf("%w: %v", ErrNoToken, lastErr)
	}
	return ErrNoToken
}

func (l *Launcher) tryFetchFingerprint(a util.Token, stats map[string]int) error {
	profileID, err := fetchFirstProfileID(a.Raw)
	if err != nil {
		l.penalize(stats, a.Raw)
		return err
	}
	if err := refreshFingerprint(profileID, a.Raw); err != nil {
		l.penalize(stats, a.Raw)
		return err
	}
	token, err := fetchProfileToken(profileID, a.Raw)
	if err != nil {
		l.penalize(stats, a.Raw)
		return err
	}
	fp, err := fetchFingerprint(profileID, a.Raw)
	if err != nil {
		l.penalize(stats, a.Raw)
		return err
	}

	l.token = token
	l.fingerprint = fp
	if fp.Navigator.Resolution != "" {
		l.resolution = fp.Navigator.Resolution
	}

	stats[a.Raw] = 0
	l.saveTokenStats(stats)
	return nil
}

// BuildProfile tạo file Preferences và orbita.config cho profile mới.
// Bỏ qua nếu là existing profile.
func (l *Launcher) BuildProfile() error {
	if !l.isNew {
		return nil
	}

	defPath := filepath.Join(l.profilePath, "Default")
	netPath := filepath.Join(defPath, "Network")
	if err := os.MkdirAll(netPath, 0o755); err != nil {
		return err
	}

	zeroData, err := storage.ReadZeroProfile()
	if err != nil {
		return fmt.Errorf("đọc zero_profile.json: %w", err)
	}

	prefs := buildPrefs(l.fingerprint, l.tz, l.ProfileID, l.ProfileID, l.ProfileName)
	prefs["profile_token"] = l.token

	zeroData["gologin"] = prefs
	if err := util.WriteJSON(filepath.Join(defPath, "Bookmarks"), zeroData); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(netPath, "Cookies"), nil, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(defPath, "Cookies"), nil, 0o644); err != nil {
		return err
	}
	if err := util.WriteJSON(filepath.Join(defPath, "Preferences"), zeroData); err != nil {
		return err
	}

	cfg := buildOrbitaConfig(prefs, l.tz, l.token)
	return util.WriteJSON(filepath.Join(l.profilePath, "orbita.config"), cfg)
}

// Launch spawn chrome.exe và chờ debug port sẵn sàng.
// onClose được gọi sau khi browser đóng (trong goroutine riêng).
func (l *Launcher) Launch(gridCfg *grid.Config, onClose func()) error {
	if _, err := os.Stat(storage.BrowserExe()); err != nil {
		return ErrBrowserNotFound
	}
	if err := l.checkLock(); err != nil {
		return err
	}
	if err := l.ensureOrbitaConfig(); err != nil {
		return err
	}
	if err := l.writePortToPrefs(); err != nil {
		return err
	}

	args := l.buildArgs(gridCfg)
	cmd, err := spawn(storage.BrowserExe(), args)
	if err != nil {
		return fmt.Errorf("spawn browser: %w", err)
	}

	go func() {
		_ = cmd.Wait()
		onClose()
	}()

	if !util.WaitForPort(l.DebugPort, 7*time.Second) {
		fmt.Printf("[WARN] Debug port %d chưa sẵn sàng sau 7s\n", l.DebugPort)
	}
	return nil
}

func (l *Launcher) checkLock() error {
	cookiePath := filepath.Join(l.profilePath, "Default", "Network", "Cookies")
	if _, err := os.Stat(cookiePath); err != nil {
		return nil // file không tồn tại — OK
	}
	f, err := os.OpenFile(cookiePath, os.O_RDWR, 0)
	if err != nil {
		return ErrProfileLocked
	}
	f.Close()
	return nil
}

func (l *Launcher) writePortToPrefs() error {
	prefsPath := filepath.Join(l.profilePath, "Default", "Preferences")
	prefs, err := util.ReadJSON(prefsPath)
	if err != nil {
		return err
	}
	if prefs["gologin"] == nil {
		prefs["gologin"] = map[string]any{}
	}
	prefs["gologin"].(map[string]any)["port"] = l.DebugPort
	return util.WriteJSON(prefsPath, prefs)
}

func (l *Launcher) buildArgs(gridCfg *grid.Config) []string {
	w, h := "1920", "1080"
	fmt.Sscanf(l.resolution, "%[^x]x%s", &w, &h)

	args := []string{
		fmt.Sprintf("--remote-debugging-port=%d", l.DebugPort),
		"--password-store=basic",
		fmt.Sprintf("--gologin-profile=%s", l.ProfileName),
		"--lang=en-US",
		"--webrtc-ip-handling-policy=default_public_interface_only",
		"--disable-features=PrintCompositorLPAC",
		fmt.Sprintf("--user-data-dir=%s", l.profilePath),
		"--restore-last-session",
	}
	if gridCfg == nil {
		args = append(args, fmt.Sprintf("--window-size=%s,%s", w, h))
	}
	return args
}

func (l *Launcher) ensureOrbitaConfig() error {
	cfgPath := filepath.Join(l.profilePath, "orbita.config")
	if _, err := os.Stat(cfgPath); err == nil {
		return nil
	}
	return l.rebuildOrbitaConfig()
}

func (l *Launcher) rebuildOrbitaConfig() error {
	prefs, err := readPrefs(l.profilePath)
	if err != nil {
		return fmt.Errorf("đọc Preferences: %w", err)
	}
	gl, _ := prefs["gologin"].(map[string]any)

	token, err := l.fetchTokenFromPrefs(gl)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrNoToken, err)
	}

	cfg := buildOrbitaConfig(gl, l.tz, token)
	return util.WriteJSON(filepath.Join(l.profilePath, "orbita.config"), cfg)
}

func (l *Launcher) fetchTokenFromPrefs(gl map[string]any) (string, error) {
	profileID, _ := gl["profile_id"].(string)
	var lastErr error
	for _, a := range l.auth {
		token, err := tryGetToken(profileID, a.Raw)
		if err != nil {
			lastErr = err
			continue
		}
		if profileID == "" {
			// cập nhật profile_id mới vào Preferences
			gl["profile_id"] = profileID
		}
		return token, nil
	}
	return "", lastErr
}

func tryGetToken(profileID, apiToken string) (string, error) {
	if profileID != "" {
		if t, err := fetchProfileToken(profileID, apiToken); err == nil {
			return t, nil
		}
	}
	// Fallback: lấy profile đầu tiên
	pid, err := fetchFirstProfileID(apiToken)
	if err != nil {
		return "", err
	}
	return fetchProfileToken(pid, apiToken)
}

// ─── Token stats ──────────────────────────────────────────────────────────────

func (l *Launcher) tokenStatsPath() string {
	return filepath.Join(storage.GologinDir(), "token_stats.json")
}

func (l *Launcher) loadTokenStats() map[string]int {
	data, err := os.ReadFile(l.tokenStatsPath())
	if err != nil {
		return map[string]int{}
	}
	var raw map[string]float64
	if err := json.Unmarshal(data, &raw); err != nil {
		return map[string]int{}
	}
	stats := make(map[string]int, len(raw))
	for k, v := range raw {
		stats[k] = int(v)
	}
	return stats
}

func (l *Launcher) saveTokenStats(stats map[string]int) {
	data, _ := json.Marshal(stats)
	_ = os.WriteFile(l.tokenStatsPath(), data, 0o644)
}

func (l *Launcher) penalize(stats map[string]int, token string) {
	stats[token]++
	l.saveTokenStats(stats)
}

// ─── GoLogin API calls ────────────────────────────────────────────────────────

func fetchTZ() (tzInfo, error) {
	var tz tzInfo
	return tz, apiGet(tzURL, nil, &tz)
}

func fetchFirstProfileID(token string) (string, error) {
	var resp struct {
		Profiles []struct{ ID string `json:"id"` } `json:"profiles"`
	}
	headers := gologinHeaders(token)
	if err := apiGet("https://api.gologin.com/browser/v2", headers, &resp); err != nil {
		return "", err
	}
	if len(resp.Profiles) == 0 {
		return "", fmt.Errorf("không có profile trong tài khoản GoLogin")
	}
	return resp.Profiles[0].ID, nil
}

func refreshFingerprint(profileID, token string) error {
	url := fmt.Sprintf("https://api.gologin.com/browser/fingerprint?os=win&osSpec=win11&template=%s", profileID)
	return apiGet(url, gologinHeaders(token), nil)
}

func fetchProfileToken(profileID, token string) (string, error) {
	url := fmt.Sprintf("https://api.gologin.com/browser/features/%s/profile-params-for-orbita-token", profileID)
	var resp struct{ Token string `json:"token"` }
	if err := apiGet(url, map[string]string{"Authorization": "Bearer " + token, "User-Agent": "Selenium-API"}, &resp); err != nil {
		return "", err
	}
	if resp.Token == "" {
		return "", fmt.Errorf("token rỗng")
	}
	return resp.Token, nil
}

func fetchFingerprint(profileID, token string) (GLProfile, error) {
	var fp GLProfile
	url := fmt.Sprintf("https://api.gologin.com/browser/%s", profileID)
	return fp, apiGet(url, gologinHeaders(token), &fp)
}

func gologinHeaders(token string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + token,
		"User-Agent":    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
	}
}

// ─── Misc helpers ─────────────────────────────────────────────────────────────

func generateName(gologinDir string) string {
	entries, _ := filepath.Glob(filepath.Join(gologinDir, "MG-*"))
	return fmt.Sprintf("P%03d", len(entries)+1)
}

func readPrefs(profilePath string) (map[string]any, error) {
	return util.ReadJSON(filepath.Join(profilePath, "Default", "Preferences"))
}

func intVal(m map[string]any, key string) int {
	v, _ := m[key].(float64)
	return int(v)
}
