//go:build windows

package main

import (
	"context"
	_ "embed"
	"os"

	"fyne.io/systray"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed zoopicon.png
var zoopIcon []byte

type ActiveTrayManager struct {
	ctx         context.Context
	nodeService *NodeService
}

func NewTrayManager() TrayManager {
	return &ActiveTrayManager{}
}

func (t *ActiveTrayManager) SetupTray(ctx context.Context, nodeService *NodeService) {
	t.ctx = ctx
	t.nodeService = nodeService

	go func() {
		systray.Run(t.onReady, t.onExit)
	}()
}

func (t *ActiveTrayManager) Quit() {
	systray.Quit()
}

func (t *ActiveTrayManager) onReady() {
	systray.SetIcon(zoopIcon)
	systray.SetTitle("Zoop One")
	systray.SetTooltip("Zoop One - Node Daemon")

	mShow := systray.AddMenuItem("Show Dashboard", "Restore Zoop One Window")
	mToggle := systray.AddMenuItem("Toggle Sharing", "Start/Stop network sharing")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Exit Application")

	t.nodeService.mu.Lock()
	sharingEnabled := t.nodeService.sharingState
	t.nodeService.mu.Unlock()

	if sharingEnabled {
		mToggle.Check()
	} else {
		mToggle.Uncheck()
	}

	// Listen for clicks in a background goroutine
	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				wailsRuntime.WindowShow(t.ctx)
			case <-mToggle.ClickedCh:
				t.nodeService.mu.Lock()
				currentState := t.nodeService.sharingState
				t.nodeService.mu.Unlock()

				newState := !currentState
				t.nodeService.ToggleSharing(newState)

				if newState {
					mToggle.Check()
				} else {
					mToggle.Uncheck()
				}
			case <-mQuit.ClickedCh:
				systray.Quit()
				wailsRuntime.Quit(t.ctx)
				os.Exit(0)
			}
		}
	}()
}

func (t *ActiveTrayManager) onExit() {
	// Cleanup operations if required on exit
}
