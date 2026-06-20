package main

import (
	"context"
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	nodeService := NewNodeService()
	trayManager := NewTrayManager()

	err := wails.Run(&options.App{
		Title:             "Zoop One",
		Width:             980,
		Height:            700,
		MinWidth:          800,
		MinHeight:         600,
		HideWindowOnClose: false, // Window closes normally, tray will handle quit on Windows
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 8, G: 12, B: 16, A: 255}, // Match --bg: hsl(220, 16%, 8%)
		OnStartup: func(ctx context.Context) {
			nodeService.Startup(ctx)
			trayManager.SetupTray(ctx, nodeService)
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
