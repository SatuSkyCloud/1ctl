// Package cluster defines the "1ctl cluster" command tree — flag names,
// input structs, and CLI wiring. Handler logic lives in handlers.go.
package cluster

import (
	"context"

	"github.com/urfave/cli/v3"
)

// --- Command tree -------------------------------------------------------

// Command returns the root cluster management command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "cluster",
		Usage: "View cluster and zone information",
		Commands: []*cli.Command{
			clusterZonesCommand(),
			clusterListCommand(),
		},
	}
}

func clusterZonesCommand() *cli.Command {
	return &cli.Command{
		Name:  "zones",
		Usage: "List available deployment zones",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleListZones(ctx)
		},
	}
}

func clusterListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all enabled clusters",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleListClusters(ctx)
		},
	}
}
