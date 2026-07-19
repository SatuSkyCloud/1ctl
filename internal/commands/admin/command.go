// Package admin defines super-admin-only operational commands.
package admin

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/urfave/cli/v3"
)

const (
	flagReason                  = "reason"
	flagExpectedUID             = "expected-uid"
	flagExpectedResourceVersion = "expected-resource-version"
	flagExpectedGeneration      = "expected-generation"
	flagRequestID               = "request-id"
	flagYes                     = "yes"
)

type deploymentAdoptInput struct {
	DeploymentID            string
	Reason                  string
	ExpectedUID             string
	ExpectedResourceVersion string
	ExpectedGeneration      int64
	RequestID               string
	Yes                     bool
}

// Command returns the super-admin command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "admin",
		Usage: "Perform guarded platform administration operations",
		Commands: []*cli.Command{
			deploymentCommand(),
		},
	}
}

func deploymentCommand() *cli.Command {
	return &cli.Command{
		Name:  "deployment",
		Usage: "Administer deployment reconciliation",
		Commands: []*cli.Command{
			deploymentAdoptCommand(),
		},
	}
}

func deploymentAdoptCommand() *cli.Command {
	var in deploymentAdoptInput
	return &cli.Command{
		Name:      "adopt",
		Usage:     "Transfer a legacy Deployment to durable reconciliation",
		ArgsUsage: "<deployment-id>",
		Flags: []cli.Flag{
			requiredString(flagReason, "Audit reason for the ownership transfer", &in.Reason, validateReason),
			requiredString(flagExpectedUID, "Exact live Deployment UID", &in.ExpectedUID, validateUUID),
			requiredString(flagExpectedResourceVersion, "Exact live Deployment resourceVersion", &in.ExpectedResourceVersion, validateNonEmpty),
			&cli.Int64Flag{
				Name:        flagExpectedGeneration,
				Usage:       "Exact live Deployment generation",
				Destination: &in.ExpectedGeneration,
				Required:    true,
				Validator:   validateGeneration,
			},
			requiredString(flagRequestID, "Stable UUID for this adoption attempt", &in.RequestID, validateUUID),
			&cli.BoolFlag{Name: flagYes, Aliases: []string{"y"}, Usage: "Skip confirmation prompt", Destination: &in.Yes},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.DeploymentID = strings.TrimSpace(cmd.Args().First())
			if err := validateUUID(in.DeploymentID); err != nil {
				return fmt.Errorf("invalid deployment ID: %w", err)
			}
			return handleDeploymentAdopt(ctx, in)
		},
	}
}

func requiredString(name, usage string, destination *string, validator func(string) error) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		Destination: destination,
		Required:    true,
		Validator:   validator,
	}
}

func validateUUID(value string) error {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || parsed == uuid.Nil {
		return fmt.Errorf("must be a non-zero UUID")
	}
	return nil
}

func validateNonEmpty(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("must not be empty")
	}
	return nil
}

func validateReason(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("must not be empty")
	}
	if len(value) > 500 {
		return fmt.Errorf("must not exceed 500 characters")
	}
	return nil
}

func validateGeneration(value int64) error {
	if value < 1 {
		return fmt.Errorf("must be at least 1")
	}
	return nil
}
