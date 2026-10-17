// Package admin defines super-admin-only operational commands.
package admin

import (
	"context"
	"fmt"
	"strings"

	"1ctl/internal/utils"
	"github.com/google/uuid"

	"github.com/urfave/cli/v3"
)

const ()

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
			machineStateRetireCommand(),
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
			deploymentRoutingAdoptCommand(),
		},
	}
}

func deploymentAdoptCommand() *cli.Command {
	var in deploymentAdoptInput
	return &cli.Command{
		Name:      "adopt",
		Usage:     "Removed: adoption is no longer served by the API",
		ArgsUsage: "<deployment-id>",
		Flags:     []cli.Flag{},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 1 {
				return invalidPositionalArgs(cmd)
			}
			in.DeploymentID = strings.TrimSpace(cmd.Args().First())
			if err := validateUUID(in.DeploymentID); err != nil {
				return fmt.Errorf("invalid deployment ID: %w", err)
			}
			return handleDeploymentAdopt(ctx, in)
		},
	}
}

func deploymentRoutingAdoptCommand() *cli.Command {
	var in deploymentAdoptInput
	return &cli.Command{
		Name:      "routing-adopt",
		Usage:     "Removed: adoption is no longer served by the API",
		ArgsUsage: "<deployment-id>",
		Flags:     []cli.Flag{},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 1 {
				return invalidPositionalArgs(cmd)
			}
			in.DeploymentID = strings.TrimSpace(cmd.Args().First())
			if err := validateUUID(in.DeploymentID); err != nil {
				return fmt.Errorf("invalid deployment ID: %w", err)
			}
			return handleDeploymentRoutingAdopt(ctx, in)
		},
	}
}

func invalidPositionalArgs(cmd *cli.Command) error {
	if err := cli.ShowSubcommandHelp(cmd); err != nil {
		return err
	}
	return fmt.Errorf("expected exactly one positional argument: %s", cmd.ArgsUsage)
}

func validateUUID(value string) error {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || parsed == uuid.Nil {
		return fmt.Errorf("must be a non-zero UUID")
	}
	return nil
}

type machineStateRetireInput struct {
	MachineID  string
	Generation int64
}

func machineStateRetireCommand() *cli.Command {
	var in machineStateRetireInput
	return &cli.Command{
		Name:      "machine-state-retire",
		Usage:     "Retire a dead-lettered machine state event so reconciliation can resume",
		ArgsUsage: "<machine-id>",
		Flags: []cli.Flag{
			&cli.Int64Flag{
				Name:        "generation",
				Usage:       "Machine generation the dead-letter belongs to",
				Required:    true,
				Destination: &in.Generation,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 1 {
				return utils.NewError("exactly one machine id is required", nil)
			}
			in.MachineID = cmd.Args().First()
			return handleMachineStateRetire(ctx, in)
		},
	}
}
