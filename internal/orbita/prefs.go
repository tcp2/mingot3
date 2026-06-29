// prefs.go — Xây dựng Preferences và orbita.config từ dữ liệu GoLogin API.
package orbita

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Các key trong fingerprint cần copy vào orbita.config (bảo vệ browser fingerprint).
var securedOpts = []string{
	"webGpu", "webgl", "webglParams", "webRTC", "webrtc",
	"mediaDevices", "plugins", "audioContext",
	"canvasMode", "canvasNoise",
	"webgl_noice_enable", "webglNoiceEnable", "webgl_noise_enable",
	"client_rects_noise_enable", "webgl_noise_value", "webglNoiseValue",
	"getClientRectsNoice", "get_client_rects_noise",
}

// buildOrbitaConfig tạo nội dung file orbita.config từ prefs và profileToken.
func buildOrbitaConfig(prefs map[string]any, tz tzInfo, profileToken string) map[string]any {
	lang := firstLang(str(prefs, "languages", "en-US"))

	gologin := map[string]any{"profile_token": profileToken}
	for _, key := range securedOpts {
		if v, ok := prefs[key]; ok {
			gologin[key] = v
		}
	}

	return map[string]any{
		"intl": map[string]any{
			"accept_languages":  lang,
			"selected_languages": lang,
		},
		"gologin": gologin,
	}
}

// buildPrefs xây dựng section "gologin" cho file Preferences của Chromium.
func buildPrefs(fp GLProfile, tz tzInfo, userID, sessionID, profileName string) map[string]any {
	w, h := parseResolution(fp.Navigator.Resolution)
	lang := fp.Navigator.Language
	noise := fp.WebGL.Mode == "noise"

	startupURLs := parseURLs(fp.StartURL)
	startupURL := ""
	if len(startupURLs) > 0 {
		startupURL = startupURLs[0]
	}

	return map[string]any{
		"_id":          sessionID,
		"profile_id":   fp.ID,
		"name":         profileName,
		"userId":       userID,
		"userPlanName": "Forever Free",

		"navigator": map[string]any{
			"platform":        fp.Navigator.Platform,
			"max_touch_points": fp.Navigator.MaxTouchPoints,
		},
		"userAgent":           fp.Navigator.UserAgent,
		"screenWidth":         w,
		"screenHeight":        h,
		"doNotTrack":          fp.Navigator.DoNotTrack,
		"hardwareConcurrency": fp.Navigator.HardwareConcurrency,
		"deviceMemory":        fp.Navigator.DeviceMemory * 1024,
		"languages":           firstLang(lang),
		"langHeader":          lang,

		"webGl": map[string]any{
			"vendor":   fp.WebGLMeta.Vendor,
			"renderer": fp.WebGLMeta.Renderer,
			"mode":     fp.WebGLMeta.Mode == "mask",
		},
		"webgl": map[string]any{
			"metadata": map[string]any{
				"vendor":   fp.WebGLMeta.Vendor,
				"renderer": fp.WebGLMeta.Renderer,
				"mode":     fp.WebGLMeta.Mode == "mask",
			},
		},
		"webgl_noice_enable":        noise,
		"webglNoiceEnable":          noise,
		"webgl_noise_enable":        noise,
		"webgl_noise_value":         fp.WebGL.Noise,
		"webglNoiseValue":           fp.WebGL.Noise,
		"getClientRectsNoice":       fp.ClientRects.Noise,
		"client_rects_noise_enable": fp.ClientRects.Mode == "noise",

		"media_devices": map[string]any{
			"enable":       fp.MediaDevices.EnableMasking,
			"uid":          fp.MediaDevices.UID,
			"audioInputs":  fp.MediaDevices.AudioInputs,
			"audioOutputs": fp.MediaDevices.AudioOutputs,
			"videoInputs":  fp.MediaDevices.VideoInputs,
		},
		"plugins": map[string]any{
			"all_enable":  fp.Plugins.EnableVulnerable,
			"flash_enable": fp.Plugins.EnableFlash,
		},
		"storage":  map[string]any{"enable": fp.Storage.Local},
		"audioContext": map[string]any{
			"enable":     fp.AudioContext.Mode != "off",
			"noiseValue": fp.AudioContext.Noise,
		},
		"canvas":     map[string]any{"mode": fp.Canvas.Mode},
		"canvasMode": fp.Canvas.Mode,
		"canvasNoise": fp.Canvas.Noise,

		"startupUrl":  startupURL,
		"startup_urls": startupURLs,

		"geolocation": map[string]any{
			"mode":      fp.Geolocation.Mode,
			"latitude":  tz.Lat,
			"longitude": tz.Lon,
			"accuracy":  tz.Accuracy,
		},
		"timezone": map[string]any{"id": tz.Timezone},
	}
}

// GLProfile là struct typed cho GoLogin API profile response.
type GLProfile struct {
	ID           string       `json:"id"`
	Navigator    Navigator    `json:"navigator"`
	WebGL        WebGLField   `json:"webGL"`
	WebGLMeta    WebGLMeta    `json:"webGLMetadata"`
	ClientRects  ClientRects  `json:"clientRects"`
	Canvas       CanvasField  `json:"canvas"`
	AudioContext AudioCtx     `json:"audioContext"`
	MediaDevices MediaDevices `json:"mediaDevices"`
	Plugins      Plugins      `json:"plugins"`
	Storage      Storage      `json:"storage"`
	Geolocation  Geolocation  `json:"geolocation"`
	StartURL     string       `json:"startUrl"`
	// Fields được copy thẳng vào orbita.config
	WebRTC    any `json:"webRTC"`
	WebGpu    any `json:"webGpu"`
	WebGLParams any `json:"webglParams"`
}

type Navigator struct {
	Platform            string  `json:"platform"`
	UserAgent           string  `json:"userAgent"`
	Language            string  `json:"language"`
	Resolution          string  `json:"resolution"`
	MaxTouchPoints      int     `json:"maxTouchPoints"`
	HardwareConcurrency int     `json:"hardwareConcurrency"`
	DeviceMemory        int     `json:"deviceMemory"`
	DoNotTrack          bool    `json:"doNotTrack"`
}
type WebGLField   struct{ Mode, Noise string }
type WebGLMeta    struct{ Vendor, Renderer, Mode string }
type ClientRects  struct{ Mode string; Noise any }
type CanvasField  struct{ Mode, Noise string }
type AudioCtx     struct{ Mode, Noise string }
type MediaDevices struct {
	EnableMasking bool   `json:"enableMasking"`
	UID           string `json:"uid"`
	AudioInputs   int    `json:"audioInputs"`
	AudioOutputs  int    `json:"audioOutputs"`
	VideoInputs   int    `json:"videoInputs"`
}
type Plugins    struct{ EnableVulnerable, EnableFlash bool }
type Storage    struct{ Local bool }
type Geolocation struct{ Mode string }

// tzInfo là timezone response từ geo.myip.link.
// ll trả về dạng "lat,lon" (string) hoặc [lat, lon] (array) tuỳ version API.
type tzInfo struct {
	Timezone string  `json:"timezone"`
	Accuracy float64 `json:"accuracy"`
	Lat      float64 // parsed từ ll
	Lon      float64 // parsed từ ll
}

func (tz *tzInfo) UnmarshalJSON(data []byte) error {
	// Dùng alias để tránh đệ quy
	type raw struct {
		Timezone string          `json:"timezone"`
		Accuracy float64         `json:"accuracy"`
		LL       json.RawMessage `json:"ll"`
	}
	var r raw
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	tz.Timezone = r.Timezone
	tz.Accuracy = r.Accuracy

	if len(r.LL) == 0 {
		return nil
	}

	// Thử array [lat, lon]
	var arr []float64
	if json.Unmarshal(r.LL, &arr) == nil && len(arr) >= 2 {
		tz.Lat, tz.Lon = arr[0], arr[1]
		return nil
	}

	// Thử string "lat,lon"
	var s string
	if json.Unmarshal(r.LL, &s) == nil {
		fmt.Sscanf(s, "%f,%f", &tz.Lat, &tz.Lon)
	}
	return nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func parseResolution(res string) (int, int) {
	var w, h int
	fmt.Sscanf(res, "%dx%d", &w, &h)
	if w == 0 {
		w = 1920
	}
	if h == 0 {
		h = 1080
	}
	return w, h
}

func firstLang(lang string) string {
	return strings.SplitN(lang, ",", 2)[0]
}

func parseURLs(raw string) []string {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func str(m map[string]any, key, def string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return def
}
