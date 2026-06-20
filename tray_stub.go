//go:build !windows

package main

import "context"

type StubTrayManager struct {
	ctx         context.Context
	nodeService *NodeService
}

func NewTrayManager() TrayManager {
	return &StubTrayManager{}
}

func (t *StubTrayManager) SetupTray(ctx context.Context, nodeService *NodeService) {
	t.ctx = ctx
	t.nodeService = nodeService
	nodeService.addLog("info", "System tray running in window-only mode on this platform.")
}

func (t *StubTrayManager) Quit() {
	// Stub implementation: No-op
}
