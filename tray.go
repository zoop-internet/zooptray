package main

import "context"

type TrayManager interface {
	SetupTray(ctx context.Context, nodeService *NodeService)
	Quit()
	IsActive() bool
}
