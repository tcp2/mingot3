// main.go — Entry point Wails + system tray.
package main

import (
	"context"
	"embed"
	"fmt"
	"os"

	"mingot/internal/storage"

	"github.com/energye/systray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsrt "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed assets
var assets embed.FS

//go:embed frontend/dist
var frontendDist embed.FS

func main() {
	// Production: profiles lưu vào %APPDATA%/Mingot/
	if appData := os.Getenv("APPDATA"); appData != "" {
		storage.SetRoot(appData + "/Mingot")
	}

	app := newApp()

	err := wails.Run(&options.App{
		Title:            "Mingot",
		Width:            1280,
		Height:           800,
		MinWidth:         960,
		MinHeight:        600,
		BackgroundColour: &options.RGBA{R: 15, G: 15, B: 26, A: 255},
		HideWindowOnClose: true,
		AssetServer: &assetserver.Options{
			Assets: frontendDist,
		},
		OnStartup: app.startup,
		Bind:      []interface{}{app},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Wails error:", err)
		os.Exit(1)
	}
}

// setupTray khởi động system tray (gọi sau khi Wails context sẵn sàng).
func setupTray(ctx context.Context) {
	icon, _ := assets.ReadFile("assets/icon.ico")

	systray.Run(func() {
		systray.SetIcon(icon)
		systray.SetTooltip("Mingot - Profile Manager")

		mShow := systray.AddMenuItem("Mở Mingot", "")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Thoát", "")

		mShow.Click(func() { wailsrt.WindowShow(ctx) })
		mQuit.Click(func() {
			systray.Quit()
			wailsrt.Quit(ctx)
		})
	}, func() {})
}
