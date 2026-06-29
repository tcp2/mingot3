// app.go — App struct bind vào Wails. Mỗi method là một endpoint frontend có thể gọi.
package main

import (
	"context"
	"errors"
	"fmt"

	"mingot/internal/profile"
)

// App là struct được Wails bind, tất cả exported method đều thành JS binding.
type App struct {
	manager *profile.Manager
}

// newApp tạo App, được gọi từ main.go.
func newApp() *App {
	return &App{manager: profile.New()}
}

// startup được Wails gọi sau khi window sẵn sàng — nhận context để emit events.
func (a *App) startup(ctx context.Context) {
	a.manager.SetContext(ctx)
}

// ─── Profile CRUD ─────────────────────────────────────────────────────────────

func (a *App) ListProfiles() apiResp[[]profile.Summary] {
	data, err := a.manager.List()
	return wrap(data, err)
}

func (a *App) CreateProfile() apiResp[*profile.CreateResult] {
	data, err := a.manager.Create()
	return wrap(data, err)
}

func (a *App) StartProfile(id string, screenW, screenH int) apiResp[*profile.StartResult] {
	data, err := a.manager.Start(id, screenW, screenH)
	return wrap(data, err)
}

func (a *App) StopProfile(id string) apiResp[any] {
	return wrap[any](nil, a.manager.Stop(id))
}

func (a *App) DeleteProfile(id string) apiResp[any] {
	return wrap[any](nil, a.manager.Delete(id))
}

func (a *App) UpdateFolder(ids []string, folder string) apiResp[any] {
	return wrap[any](nil, a.manager.UpdateFolder(ids, folder))
}

// ImportProfile nhận ZIP bytes từ frontend (Uint8Array → []byte tự động bởi Wails).
func (a *App) ImportProfile(zipData []byte, overwrite bool) apiResp[*profile.ImportResult] {
	data, err := a.manager.Import(zipData, overwrite)
	return wrap(data, err)
}

// ExportProfile trả về ZIP bytes để frontend trigger download.
func (a *App) ExportProfile(id string) apiResp[[]byte] {
	data, err := a.manager.Export(id)
	return wrap(data, err)
}

func (a *App) ArrangeWindows(screenW, screenH int) apiResp[*profile.ArrangeResult] {
	data, err := a.manager.ArrangeWindows(screenW, screenH)
	return wrap(data, err)
}

// ─── Response wrapper ─────────────────────────────────────────────────────────

// apiResp là envelope JSON thống nhất trả về frontend.
type apiResp[T any] struct {
	Success   bool   `json:"success"`
	Data      T      `json:"data,omitempty"`
	Error     string `json:"error,omitempty"`
	Code      string `json:"code,omitempty"`
	ProfileID string `json:"profileId,omitempty"`
}

func wrap[T any](data T, err error) apiResp[T] {
	if err == nil {
		return apiResp[T]{Success: true, Data: data}
	}

	resp := apiResp[T]{Error: err.Error()}
	var ae *profile.AppError
	if errors.As(err, &ae) {
		resp.Code = string(ae.Code)
		resp.ProfileID = ae.ProfileID
	}
	fmt.Printf("[APP] Error: %v\n", err)
	return resp
}
