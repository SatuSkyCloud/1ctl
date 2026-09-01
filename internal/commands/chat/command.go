// Package chat defines the "1ctl chat" command tree. The interactive chat
// engine itself lives in internal/chat; this package is the thin CLI wiring
// (flags, input struct, handlers).
package chat

import (
	"context"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const (
	flagProvider = "provider"
	flagModel    = "model"
	flagShowKey  = "show-key"
)

// --- Input structs ------------------------------------------------------

// chatInput mirrors the command flags. It is populated by urfave/cli via
// Destination bindings and consumed by the handlers.
type chatInput struct {
	Provider string // --provider: openai, claude or deepseek
	Model    string // --model: override the provider's default model
	ShowKey  bool   // --show-key: show the configured API key prefix
}

// --- Command tree -------------------------------------------------------

// Command returns the root chat command tree.
func Command() *cli.Command {
	var in chatInput
	return &cli.Command{
		Name:  "chat",
		Usage: "Start an interactive chat with an AI provider (OpenAI, Claude, DeepSeek)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagProvider,
				Usage:       "AI provider to use: openai, claude or deepseek",
				Destination: &in.Provider,
			},
			&cli.StringFlag{
				Name:        flagModel,
				Usage:       "Model to use (defaults to the provider's default model)",
				Destination: &in.Model,
			},
			&cli.BoolFlag{
				Name:        flagShowKey,
				Usage:       "Show the configured API key prefix (never the full key)",
				Destination: &in.ShowKey,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleChat(ctx, cmd, in)
		},
	}
}
