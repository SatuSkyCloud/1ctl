// Package issuer defines the "1ctl issuer" command tree.
package issuer

import (
	"context"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const (
	flagDeploymentID = "deployment-id"
	flagEmail        = "email"
	flagEnvironment  = "environment"
	flagIssuerID     = "issuer-id"
)

// --- Input structs ------------------------------------------------------

type issuerCreateInput struct {
	DeploymentID string
	Email        string
	Environment  string
}

type issuerDeleteInput struct {
	IssuerID string
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

func optionalString(name, usage string, dest *string, value string, validate func(string) error) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		Destination: dest,
		Value:       value,
		Validator:   validate,
	}
}

// --- Command tree -------------------------------------------------------

// Command returns the root issuer command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:   "issuer",
		Usage:  "Manage cert-manager issuers (internal — TLS is automatic when adding a custom domain)",
		Hidden: true,
		Commands: []*cli.Command{
			issuerCreateCommand(),
			issuerListCommand(),
			issuerDeleteCommand(),
		},
	}
}

func issuerCreateCommand() *cli.Command {
	var in issuerCreateInput
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new certificate issuer",
		Flags: []cli.Flag{
			requiredString(flagDeploymentID, "Deployment ID", &in.DeploymentID, nil),
			requiredString(flagEmail, "Email address for Let's Encrypt notifications", &in.Email, nil),
			optionalString(flagEnvironment, "Environment (production/staging)", &in.Environment, "staging", nil),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error { return handleCreateIssuer(ctx, in) },
	}
}

func issuerListCommand() *cli.Command {
	return &cli.Command{
		Name:   "list",
		Usage:  "List all certificate issuers",
		Action: func(ctx context.Context, cmd *cli.Command) error { return handleListIssuers(ctx) },
	}
}

func issuerDeleteCommand() *cli.Command {
	var in issuerDeleteInput
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a certificate issuer",
		Flags: []cli.Flag{
			requiredString(flagIssuerID, "Issuer ID to delete", &in.IssuerID, nil),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error { return handleDeleteIssuer(ctx, in) },
	}
}
