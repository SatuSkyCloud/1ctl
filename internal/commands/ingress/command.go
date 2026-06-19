// Package ingress defines the "1ctl ingress" command tree.
package ingress

import (
	"context"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const (
	flagDeploymentID = "deployment-id"
	flagServiceID    = "service-id"
	flagAppLabel     = "app-label"
	flagNamespace    = "namespace"
	flagDomain       = "domain"
	flagPort         = "port"
	flagCustomDNS    = "custom-dns"
	flagIngressID    = "ingress-id"
	flagYes          = "yes"
)

// --- Input structs ------------------------------------------------------

type ingressUpsertInput struct {
	DeploymentID string
	ServiceID    string
	AppLabel     string
	Namespace    string
	Domain       string
	Port         int
	CustomDNS    bool
}

type ingressDeleteInput struct {
	IngressID string
	Yes       bool
}

// --- Flag constructors --------------------------------------------------

func requiredString(name, usage string, dest *string, validate func(string) error) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		Destination: dest,
		Required:    true,
		Validator:   validate,
	}
}

// --- Command tree -------------------------------------------------------

// Command returns the root ingress command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:   "ingress",
		Usage:  "Create or update an ingress for a deployment (low-level — prefer `1ctl domains`)",
		Hidden: true,
		Commands: []*cli.Command{
			ingressListCommand(),
			ingressDeleteCommand(),
		},
		Action: ingressUpsertAction,
	}
}

func ingressUpsertAction(ctx context.Context, cmd *cli.Command) error {
	if !cmd.IsSet("deployment-id") && !cmd.IsSet("domain") && cmd.NArg() == 0 {
		return cli.ShowCommandHelp(ctx, cmd, "ingress")
	}
	if cmd.NArg() > 0 {
		return cli.ShowSubcommandHelp(cmd)
	}
	var in ingressUpsertInput
	in.DeploymentID = cmd.String("deployment-id")
	in.ServiceID = cmd.String("service-id")
	in.AppLabel = cmd.String("app-label")
	in.Namespace = cmd.String("namespace")
	in.Domain = cmd.String("domain")
	in.Port = cmd.Int("port")
	in.CustomDNS = cmd.Bool("custom-dns")
	return handleUpsertIngress(ctx, in)
}

func ingressListCommand() *cli.Command {
	return &cli.Command{
		Name:   "list",
		Usage:  "List all ingresses",
		Action: func(ctx context.Context, cmd *cli.Command) error { return handleListIngresses(ctx) },
	}
}

func ingressDeleteCommand() *cli.Command {
	var in ingressDeleteInput
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete an ingress",
		Flags: []cli.Flag{
			requiredString(flagIngressID, "Ingress ID to delete", &in.IngressID, nil),
			&cli.BoolFlag{
				Name:        flagYes,
				Aliases:     []string{"y"},
				Usage:       "Skip confirmation prompt",
				Destination: &in.Yes,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error { return handleDeleteIngress(ctx, in) },
	}
}
