// Package util cung cấp các helper dùng chung: ULID, port, JSON, GoLogin tokens.
package util

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ─── ULID ─────────────────────────────────────────────────────────────────────

// crockford base32 alphabet — chuẩn của ULID spec.
const crockfordAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// NewULID tạo một ULID mới (monotonic, sortable by time).
func NewULID() string {
	ts := uint64(time.Now().UnixMilli())
	rnd := make([]byte, 10)
	_, _ = rand.Read(rnd)

	var buf [26]byte

	// 10 chars timestamp (48-bit, 5 bits/char)
	for i := 9; i >= 0; i-- {
		buf[i] = crockfordAlphabet[ts&0x1F]
		ts >>= 5
	}

	// 16 chars random (80-bit, 5 bits/char) — xử lý 5 bytes mỗi lần
	encodeRandChunk := func(b []byte, out []byte) {
		v := uint64(b[0])<<32 | uint64(b[1])<<24 | uint64(b[2])<<16 | uint64(b[3])<<8 | uint64(b[4])
		for i := len(out) - 1; i >= 0; i-- {
			out[i] = crockfordAlphabet[v&0x1F]
			v >>= 5
		}
	}
	encodeRandChunk(rnd[:5], buf[10:18])
	encodeRandChunk(rnd[5:], buf[18:26])

	return string(buf[:])
}

// ─── Port ─────────────────────────────────────────────────────────────────────

// FreePort trả về một port TCP trống trên localhost.
func FreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}

// WaitForPort poll cổng HTTP cho đến khi OK hoặc hết timeout.
func WaitForPort(port int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	url := fmt.Sprintf("http://127.0.0.1:%d/json/version", port)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

// ─── JSON ─────────────────────────────────────────────────────────────────────

// ReadJSON đọc file JSON vào map.
func ReadJSON(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	return out, json.Unmarshal(data, &out)
}

// WriteJSON ghi map ra file JSON (compact).
func WriteJSON(path string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ─── GoLogin Tokens ───────────────────────────────────────────────────────────

// Token là JWT token và sub (userId) của tài khoản GoLogin.
type Token struct {
	Raw    string
	UserID string
}

var jwtRe = regexp.MustCompile(`eyJ[a-zA-Z0-9_-]+\.eyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+`)

// ExtractGoLoginTokens quét LevelDB files của GoLogin, trích xuất tất cả JWT.
func ExtractGoLoginTokens(storeDirs []string) []Token {
	seen := map[string]bool{}
	var tokens []Token

	for _, dir := range storeDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".log") && !strings.HasSuffix(e.Name(), ".ldb") {
				continue
			}
			found := extractFromFile(filepath.Join(dir, e.Name()), seen)
			tokens = append(tokens, found...)
		}
	}
	return tokens
}

func extractFromFile(path string, seen map[string]bool) []Token {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var tokens []Token
	for _, raw := range jwtRe.FindAllString(string(data), -1) {
		if seen[raw] {
			continue
		}
		userID, err := jwtSub(raw)
		if err != nil || userID == "" {
			continue
		}
		seen[raw] = true
		tokens = append(tokens, Token{Raw: raw, UserID: userID})
	}
	return tokens
}

// jwtSub trích xuất trường "sub" từ JWT payload mà không cần verify signature.
func jwtSub(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid jwt")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", err
	}
	sub, _ := claims["sub"].(string)
	return sub, nil
}
