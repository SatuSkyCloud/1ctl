// Package marketplace defines the "1ctl marketplace" command tree.
package marketplace

import (
	"context"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const (
	flagHostname     = "hostname"
	flagCPU          = "cpu"
	flagMemory       = "memory"
	flagDomain       = "domain"
	flagStorageSize  = "storage-size"
	flagLimit        = "limit"
	flagOffset       = "offset"
	flagSort         = "sort"
)

// --- Input structs ------------------------------------------------------

type marketplaceListInput struct {
	Limit  int
	Offset int
	Sort   string
}

type marketplaceDeployInput struct {
	AppName     string   // positional: marketplace app name or ID
	DeployName  string   // optional positional: deployment name (defaults to app name)
	Hostnames   []string
	CPU         string
	Memory      string
	Domain      string
	StorageSize string
}

// --- Command tree -------------------------------------------------------

// Command returns the root marketplace command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:    "marketplace",
		Aliases: []string{"market", "apps"},
		Usage:   "Browse and deploy marketplace apps",
		Commands: []*cli.Command{
			marketplaceListCommand(),
			marketplaceGetCommand(),
			marketplaceDeployCommand(),
		},
	}
}

func marketplaceListCommand() *cli.Command {
	var in marketplaceListInput
	return &cli.Command{
		Name:  "list",
		Usage: "List marketplace apps",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        flagLimit,
				Usage:       "Number of apps to show",
				Destination: &in.Limit,
				Value:       25,
			},
			&cli.IntFlag{
				Name:        flagOffset,
				Usage:       "Offset for pagination",
				Destination: &in.Offset,
				Value:       0,
			},
			&cli.StringFlag{
				Name:        flagSort,
				Usage:       "Sort by field (e.g., popularity, name)",
				Destination: &in.Sort,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleMarketplaceList(ctx, in)
		},
	}
}

func marketplaceGetCommand() *cli.Command {
	return &cli.Command{
		Name:      "get",
		Usage:     "Show details for a marketplace app",
		ArgsUsage: "<app>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			return handleMarketplaceGet(ctx, cmd.Args().First())
		},
	}
}

func marketplaceDeployCommand() *cli.Command {
	return &cli.Command{
		Name:      "deploy",
		Usage:     "Deploy a marketplace app",
		ArgsUsage: "<app> [deployment-name]",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:        flagHostname,
				Usage:       "Machine hostname(s) to deploy to",
			},
			&cli.StringFlag{
				Name:        flagCPU,
				Usage:       "CPU cores allocation (e.g., '2')",
			},
			&cli.StringFlag{
				Name:        flagMemory,
				Usage:       "Memory allocation (e.g., '4Gi')",
			},
			&cli.StringFlag{
				Name:        flagDomain,
				Usage:       "Custom domain (default: auto-generated)",
			},
			&cli.StringFlag{
				Name:        flagStorageSize,
				Usage:       "Storage size for persistent data (e.g., '10Gi')",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in := marketplaceDeployInput{
				AppName:     cmd.Args().First(),
				DeployName:  cmd.Args().Get(1),
				Hostnames:   cmd.StringSlice(flagHostname),
				CPU:         cmd.String(flagCPU),
				Memory:      cmd.String(flagMemory),
				Domain:      cmd.String(flagDomain),
				StorageSize: cmd.String(flagStorageSize),
			}
			return handleMarketplaceDeploy(ctx, in)
		},
	}
}
