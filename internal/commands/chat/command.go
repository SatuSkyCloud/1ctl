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
	Provider string // --provider: openai, claude or deepseek (default: active provider)
	Model    string // --model: override the provider's default model for this session
	ShowKey  bool   // --show-key: echo the API key while typing during /connect (default: hidden)
}

// --- Command tree -------------------------------------------------------

// Command returns the root chat command tree.
func Command() *cli.Command {
	var in chatInput
	return &cli.Command{
		Name:      "chat",
		Usage:     "Interactive AI copilot for SatuSky Cloud (OpenAI, Claude or DeepSeek)",
		ArgsUsage: "[message...]",
		Description: `An interactive developer copilot: ask questions and get SatuSky best
practices (advisor), provision Postgres/Valkey/NATS, domains, env/secrets and
deploys through the real 1ctl (operator), and scaffold projects with file and
shell tools (builder).

With no arguments an interactive chat starts (connect a provider on first
run by pasting an API key — input hidden and tested before "Connected").
Positional arguments run a single turn instead:

    1ctl chat "how do I do a zero-downtime deploy?"

SatuSky actions require ` + "`1ctl auth login`" + `; questions and file work
don't need it. Keys may also come from OPENAI_API_KEY,
ANTHROPIC_API_KEY or DEEPSEEK_API_KEY. See /help inside the chat for the
slash commands.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagProvider,
				Usage:       "AI provider to use: openai, claude or deepseek (default: the active provider)",
				Destination: &in.Provider,
			},
			&cli.StringFlag{
				Name:        flagModel,
				Usage:       "Model for this session (default: the provider's default model)",
				Destination: &in.Model,
			},
			&cli.BoolFlag{
				Name:        flagShowKey,
				Usage:       "Echo the API key while typing during /connect (default: hidden input)",
				Destination: &in.ShowKey,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleChat(ctx, cmd, in)
		},
	}
}
