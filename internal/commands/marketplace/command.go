// Package marketplace defines the "1ctl marketplace" command tree.
package marketplace

import (
	"context"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const (
	flagMarketplaceID    = "marketplace-id"
	flagName             = "name"
	flagHostname         = "hostname"
	flagCPU              = "cpu"
	flagMemory           = "memory"
	flagDomain           = "domain"
	flagStorageSize      = "storage-size"
	flagStorageClass     = "storage-class"
	flagMulticluster     = "multicluster"
	flagMulticlusterMode = "multicluster-mode"
	flagLimit            = "limit"
	flagOffset           = "offset"
	flagSort             = "sort"
)

// --- Flag constructors --------------------------------------------------
// Every constructor requires Destination — you cannot create a flag
// without wiring it to a struct field.

func requiredString(name, usage string, dest *string, validate func(string) error) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		Destination: dest,
		Required:    true,
		Validator:   validate,
	}
}

func optionalString(name, usage string, dest *string, validate func(string) error) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		Destination: dest,
		Validator:   validate,
	}
}

// --- Input structs ------------------------------------------------------

type marketplaceListInput struct {
	Limit  int
	Offset int
	Sort   string
}

type marketplaceGetInput struct {
	MarketplaceID string
}

type marketplaceDeployInput struct {
	MarketplaceID   string
	Name            string
	Hostnames      []string
	CPU            string
	Memory         string
	Domain         string
	StorageSize    string
	StorageClass   string
	Multicluster   bool
	MulticlusterMode string
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
				Value:       20,
			},
			&cli.IntFlag{
				Name:        flagOffset,
				Usage:       "Offset for pagination",
				Destination: &in.Offset,
				Value:       0,
			},
			optionalString(flagSort, "Sort by field (e.g., popularity, name)", &in.Sort, nil),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleMarketplaceList(ctx, in)
		},
	}
}

func marketplaceGetCommand() *cli.Command {
	var in marketplaceGetInput
	return &cli.Command{
		Name:  "get",
		Usage: "Get marketplace app details",
		Flags: []cli.Flag{
			requiredString(flagMarketplaceID, "Marketplace app ID", &in.MarketplaceID, nil),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleMarketplaceGet(ctx, in)
		},
	}
}

func marketplaceDeployCommand() *cli.Command {
	var in marketplaceDeployInput
	return &cli.Command{
		Name:  "deploy",
		Usage: "Deploy a marketplace app",
		Flags: []cli.Flag{
			requiredString(flagMarketplaceID, "Marketplace app ID", &in.MarketplaceID, nil),
			requiredString(flagName, "Deployment name", &in.Name, nil),
			&cli.StringSliceFlag{
				Name:        flagHostname,
				Usage:       "Machine hostname(s) to deploy to",
				Destination: &in.Hostnames,
			},
			optionalString(flagCPU, "CPU cores allocation (e.g., '2')", &in.CPU, nil),
			optionalString(flagMemory, "Memory allocation (e.g., '4Gi')", &in.Memory, nil),
			optionalString(flagDomain, "Custom domain (default: auto-generated)", &in.Domain, nil),
			optionalString(flagStorageSize, "Storage size for PVC (e.g., '10Gi')", &in.StorageSize, nil),
			optionalString(flagStorageClass, "Storage class for PVC", &in.StorageClass, nil),
			&cli.BoolFlag{
				Name:        flagMulticluster,
				Usage:       "Enable multi-cluster deployment",
				Destination: &in.Multicluster,
			},
			optionalString(flagMulticlusterMode, "Multi-cluster mode: 'active-active' or 'active-passive'", &in.MulticlusterMode, nil),
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			// Set default value for multicluster-mode
			if in.MulticlusterMode == "" {
				in.MulticlusterMode = "active-passive"
			}
			return ctx, nil
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleMarketplaceDeploy(ctx, in)
		},
	}
}
