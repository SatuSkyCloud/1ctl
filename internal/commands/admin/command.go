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
			deploymentMoveCommand(),
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

type deploymentMoveInput struct {
	App          string
	DeploymentID string
	From         string
	To           string
	Yes          bool
}

func deploymentMoveCommand() *cli.Command {
	var in deploymentMoveInput
	return &cli.Command{
		Name:  "move",
		Usage: "Move a running app from one machine to another with a rolling update",
		Description: "Swaps one machine for another. The target gets the same checks as automatic\n" +
			"placement, and the old machine is released only after the new pods are Ready.\n" +
			"Only stateless renter apps with no rollout in progress can move.\n\n" +
			"Example:\n   1ctl admin deployment move ims-dev --from compute-main-04 --to compute-main-01",
		ArgsUsage: "<app>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "deployment-id", Usage: "Deployment ID (instead of the app name)", Destination: &in.DeploymentID},
			&cli.StringFlag{Name: "from", Usage: "Machine name the app runs on now", Required: true, Destination: &in.From},
			&cli.StringFlag{Name: "to", Usage: "Machine name to move the app to", Required: true, Destination: &in.To},
			&cli.BoolFlag{Name: "yes", Usage: "Skip the confirmation prompt", Destination: &in.Yes},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() > 1 || (cmd.Args().Len() == 0 && in.DeploymentID == "") {
				return invalidPositionalArgs(cmd)
			}
			in.App = strings.TrimSpace(cmd.Args().First())
			in.DeploymentID = strings.TrimSpace(in.DeploymentID)
			if in.DeploymentID != "" {
				if err := validateUUID(in.DeploymentID); err != nil {
					return fmt.Errorf("invalid deployment ID: %w", err)
				}
			}
			in.From, in.To = strings.TrimSpace(in.From), strings.TrimSpace(in.To)
			if in.From == "" || in.To == "" || in.From == in.To {
				return fmt.Errorf("--from and --to must name two different machines")
			}
			return handleDeploymentMove(ctx, in)
		},
	}
}
