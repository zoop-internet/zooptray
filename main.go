package main

import (
	"context"
	"embed"
	_ "embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed zoopicon.png
var appIcon []byte

func main() {
	nodeService := NewNodeService()
	trayManager := NewTrayManager()

	err := wails.Run(&options.App{
		Title:             "Zoop One",
		Width:             1100,
		Height:            760,
		MinWidth:          800,
		MinHeight:         600,
		HideWindowOnClose: false,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 8, G: 12, B: 16, A: 255},
		Linux: &linux.Options{
			Icon: appIcon,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHiddenInset(),
			Appearance:           mac.DefaultAppearance,
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			About: &mac.AboutInfo{
				Title:   "Zoop One",
				Message: "A decentralized network sharing node.\n\nCopyright © 2024 Zoop Network",
				Icon:    appIcon,
			},
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
		OnStartup: func(ctx context.Context) {
			wailsRuntime.WindowCenter(ctx)
			nodeService.Startup(ctx)
			trayManager.SetupTray(ctx, nodeService)
		},
		OnBeforeClose: func(ctx context.Context) bool {
			if trayManager.IsActive() {
				wailsRuntime.WindowHide(ctx)
				return true
			}
			return false
		},
		OnShutdown: func(ctx context.Context) {
			nodeService.Shutdown(ctx)
			trayManager.Quit()
		},
		Bind: []interface{}{
			nodeService,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
