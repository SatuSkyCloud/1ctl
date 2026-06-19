// Package audit defines the "1ctl audit" command tree.
package audit

import (
	"context"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const (
	flagID     = "id"
	flagLimit  = "limit"
	flagAction = "action"
	flagUser   = "user"
)

// --- Flag constructors --------------------------------------------------
// Every constructor requires Destination.

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

type auditListInput struct {
	Limit  int
	Action string
	User   string
}

type auditGetInput struct {
	ID string
}

// --- Command tree -------------------------------------------------------

// Command returns the root audit command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "audit",
		Usage: "View audit logs",
		Commands: []*cli.Command{
			auditListCommand(),
			auditGetCommand(),
		},
	}
}

func auditListCommand() *cli.Command {
	var in auditListInput
	return &cli.Command{
		Name:  "list",
		Usage: "List audit logs",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        flagLimit,
				Usage:       "Number of logs to show",
				Destination: &in.Limit,
				Value:       20,
			},
			optionalString(flagAction, "Filter by action type", &in.Action, nil),
			optionalString(flagUser, "Filter by user ID", &in.User, nil),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleAuditList(ctx, in)
		},
	}
}

func auditGetCommand() *cli.Command {
	var in auditGetInput
	return &cli.Command{
		Name:  "get",
		Usage: "Get audit log details",
		Flags: []cli.Flag{
			requiredString(flagID, "Audit log ID", &in.ID, nil),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleAuditGet(ctx, in)
		},
	}
}
