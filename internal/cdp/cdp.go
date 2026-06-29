// Package cdp cung cấp các thao tác Chrome DevTools Protocol qua WebSocket.
package cdp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// msg là cấu trúc CDP request/response tối giản.
type msg struct {
	ID     int            `json:"id"`
	Method string         `json:"method,omitempty"`
	Params map[string]any `json:"params,omitempty"`
	Result map[string]any `json:"result,omitempty"`
}

// dial kết nối WebSocket tới target đầu tiên có type="page" tại port.
func dial(port int) (*websocket.Conn, error) {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/json/list", port))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var targets []map[string]any
	if err := json.Unmarshal(body, &targets); err != nil {
		return nil, err
	}

	for _, t := range targets {
		if t["type"] != "page" {
			continue
		}
		wsURL, _ := t["webSocketDebuggerUrl"].(string)
		if wsURL == "" {
			continue
		}
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		return conn, err
	}
	return nil, fmt.Errorf("no page target at port %d", port)
}

// dialVersion kết nối WebSocket qua /json/version (dùng cho Browser.close).
func dialVersion(port int) (*websocket.Conn, error) {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/json/version", port))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	wsURL, _ := info["webSocketDebuggerUrl"].(string)
	if wsURL == "" {
		return nil, fmt.Errorf("no webSocketDebuggerUrl")
	}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	return conn, err
}

func send(conn *websocket.Conn, m msg) (msg, error) {
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if err := conn.WriteJSON(m); err != nil {
		return msg{}, err
	}
	var resp msg
	return resp, conn.ReadJSON(&resp)
}

// CloseBrowser gửi lệnh Browser.close tới trình duyệt tại port.
func CloseBrowser(port int) error {
	conn, err := dialVersion(port)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = send(conn, msg{ID: 1, Method: "Browser.close"})
	return err
}

// SetWindowBounds di chuyển và resize cửa sổ trình duyệt qua CDP.
func SetWindowBounds(port, x, y, w, h int) error {
	conn, err := dial(port)
	if err != nil {
		return err
	}
	defer conn.Close()

	r1, err := send(conn, msg{ID: 1, Method: "Browser.getWindowForTarget"})
	if err != nil {
		return err
	}
	winID, ok := r1.Result["windowId"]
	if !ok {
		return fmt.Errorf("windowId not found")
	}

	_, err = send(conn, msg{
		ID:     2,
		Method: "Browser.setWindowBounds",
		Params: map[string]any{
			"windowId": winID,
			"bounds": map[string]any{
				"left": x, "top": y,
				"width": w, "height": h,
				"windowState": "normal",
			},
		},
	})
	return err
}
