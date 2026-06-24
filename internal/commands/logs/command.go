// Package logs defines the "1ctl logs" command tree — flag names,
// input structs, and CLI wiring.  Handler logic lives in handlers.go.
package logs

import (
	"context"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const (
	flagDeploymentID = "deployment-id"
	flagApp          = "app"
	flagConfig       = "config"
	flagTail         = "tail"
	flagNamespace    = "namespace"
	flagBatchSize    = "batch-size"
)

// --- Input structs ------------------------------------------------------

type logsInput struct {
	DeploymentID string
	App          string
	Config       string
	Tail         int
}

type logsStreamInput struct {
	DeploymentID string
	Namespace    string
	App          string
	BatchSize    int
	Config       string
}

// Package-level binding targets for the root command.
// Root flags are bound via Destination and then shallow-copied in the
// action closure so each invocation gets its own value.
var rootLogsIn logsInput

// --- Flag constructors --------------------------------------------------

func optionalString(name, usage string, dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		Destination: dest,
	}
}

func optionalStringWithAliases(name, usage string, aliases []string, dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        name,
		Aliases:     aliases,
		Usage:       usage,
		Destination: dest,
	}
}

func optionalInt(name, usage string, dest *int, defaultValue int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        name,
		Usage:       usage,
		Destination: dest,
		Value:       defaultValue,
	}
}

func optionalIntWithAliases(name, usage string, aliases []string, dest *int, defaultValue int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        name,
		Aliases:     aliases,
		Usage:       usage,
		Destination: dest,
		Value:       defaultValue,
	}
}

// --- Command tree -------------------------------------------------------

// Command returns the root logs command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "logs",
		Usage: "View application logs",
		Flags: []cli.Flag{
			optionalStringWithAliases(flagDeploymentID, "Deployment ID to view logs for", []string{"d"}, &rootLogsIn.DeploymentID),
			optionalString(flagApp, "App name to resolve (alternative to --deployment-id)", &rootLogsIn.App),
			optionalString(flagConfig, "Config name or path (e.g. staging, satusky.staging.toml). Default: satusky.toml", &rootLogsIn.Config),
			optionalIntWithAliases(flagTail, "Number of lines to show (default: 100)", []string{"n"}, &rootLogsIn.Tail, 100),
		},
		Commands: []*cli.Command{
			logsStreamCommand(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// Shallow copy so the handler isn't racing with future flag parsing.
			in := rootLogsIn
			return handleLogs(ctx, in)
		},
	}
}

func logsStreamCommand() *cli.Command {
	var in logsStreamInput
	return &cli.Command{
		Name:  "stream",
		Usage: "Stream live application logs",
		Flags: []cli.Flag{
			optionalStringWithAliases(flagDeploymentID, "Deployment ID (resolves namespace and app label automatically)", []string{"d"}, &in.DeploymentID),
			optionalStringWithAliases(flagNamespace, "Kubernetes namespace (use with --app)", []string{"n"}, &in.Namespace),
			optionalStringWithAliases(flagApp, "App label (use with --namespace)", []string{"a"}, &in.App),
			optionalInt(flagBatchSize, "Log lines per batch sent by the server", &in.BatchSize, 100),
			optionalString(flagConfig, "Config name or path (e.g. staging, satusky.staging.toml). Default: satusky.toml", &in.Config),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleLogsStream(ctx, in)
		},
	}
}
