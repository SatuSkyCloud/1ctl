package chat

import (
	"context"
	"strings"

	chatengine "1ctl/internal/chat"

	"github.com/urfave/cli/v3"
)

// handleChat wires the "1ctl chat" command to the interactive chat engine.
// Any positional args run a single turn (`1ctl chat "prompt"`); with no
// args the interactive REPL starts.
func handleChat(ctx context.Context, cmd *cli.Command, in chatInput) error {
	opts := chatengine.ReplOptions{
		Provider: chatengine.Provider(in.Provider),
		Model:    in.Model,
		ShowKey:  in.ShowKey,
	}
	if args := cmd.Args().Slice(); len(args) > 0 {
		opts.OneShot = strings.Join(args, " ")
	}
	return chatengine.Run(ctx, opts)
}
