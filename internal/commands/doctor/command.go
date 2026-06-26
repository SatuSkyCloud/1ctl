// Package doctor defines the "1ctl doctor" command tree.
package doctor

import (
	"context"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const (
	flagDeploymentID = "deployment-id"
	flagApp          = "app"
	flagConfig       = "config"
	flagHealthPath   = "health-path"
	flagSmoke        = "smoke"
)

// --- Input structs ------------------------------------------------------

type doctorInput struct {
	DeploymentID   string
	App            string
	Config         string
	HealthPath     string
	HealthPathSet  bool  // true when --health-path was explicitly passed
	Smoke          bool
}

// --- Command tree -------------------------------------------------------

// Command returns the root doctor command tree.
func Command() *cli.Command {
	var in doctorInput
	return &cli.Command{
		Name:  "doctor",
		Usage: "Diagnose auth, backend reachability, zones, clusters, and live deployments",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagDeploymentID,
				Aliases:     []string{"d"},
				Usage:       "Check one deployment instead of the whole namespace",
				Destination: &in.DeploymentID,
			},
			&cli.StringFlag{
				Name:        flagApp,
				Usage:       "App name to resolve (alternative to --deployment-id)",
				Destination: &in.App,
			},
			&cli.StringFlag{
				Name:        flagConfig,
				Usage:       "Config name or path (e.g. staging, satusky.staging.toml). Used to resolve a deployment ID when not provided.",
				Destination: &in.Config,
			},
			&cli.StringFlag{
				Name:        flagHealthPath,
				Usage:       "Explicit HTTP path for app-level smoke success (default: tries /health then /). Enforces 2xx/3xx success.",
				Destination: &in.HealthPath,
			},
			&cli.BoolFlag{
				Name:        flagSmoke,
				Usage:       "Run smoke checks across all namespace deployments (opt-in for namespace-wide doctor)",
				Destination: &in.Smoke,
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			in.HealthPathSet = cmd.IsSet(flagHealthPath)
			return ctx, nil
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleDoctor(ctx, in)
		},
	}
}
