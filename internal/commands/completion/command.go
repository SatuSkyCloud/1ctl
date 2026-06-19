// Package completion defines the "1ctl completion" command tree.
package completion

import (
	"context"
	"io"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const flagAppName = "app-name"

// --- Input structs ------------------------------------------------------

type completionWriterInput struct {
	Writer io.Writer
	Name   string
}

// --- Command tree -------------------------------------------------------

// Command returns the root completion command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "completion",
		Usage: "Generate and install shell completion scripts",
		Commands: []*cli.Command{
			{
				Name:  "install",
				Usage: "Auto-detect shell and install completion (add one line to shell config)",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return handleCompletionInstall(ctx, cmd.Root().Writer)
				},
			},
			{
				Name:  "bash",
				Usage: "Generate bash completion script",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return handleBashCompletion(ctx, completionWriterInput{
						Writer: cmd.Root().Writer,
						Name:   cmd.Root().Name,
					})
				},
			},
			{
				Name:  "zsh",
				Usage: "Generate zsh completion script",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return handleZshCompletion(ctx, completionWriterInput{
						Writer: cmd.Root().Writer,
						Name:   cmd.Root().Name,
					})
				},
			},
			{
				Name:  "fish",
				Usage: "Generate fish completion script",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return handleFishCompletion(ctx, completionWriterInput{
						Writer: cmd.Root().Writer,
						Name:   cmd.Root().Name,
					})
				},
			},
			{
				Name:  "powershell",
				Usage: "Generate PowerShell completion script",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return handlePowerShellCompletion(ctx, completionWriterInput{
						Writer: cmd.Root().Writer,
						Name:   cmd.Root().Name,
					})
				},
			},
		},
	}
}
