// api.go — HTTP helper nội bộ cho orbita package.
package orbita

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var client = &http.Client{Timeout: 15 * time.Second}

// apiGet thực hiện GET request, decode JSON vào dest (nil để bỏ qua).
func apiGet(url string, headers map[string]string, dest any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		preview := string(body)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, preview)
	}
	if dest == nil {
		return nil
	}
	return json.Unmarshal(body, dest)
}
