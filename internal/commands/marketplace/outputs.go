package marketplace

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/urfave/cli/v3"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
)

const flagReveal = "reveal"

// marketplaceOutputsCommand lists the outputs a marketplace deployment's
// package declares and reveals one on request. For example a SiYuan instance
// opens on an access-code lock screen; its code is the access-auth-code output.
func marketplaceOutputsCommand() *cli.Command {
	return &cli.Command{
		Name:      "outputs",
		Usage:     "List or reveal a marketplace deployment's app credentials and outputs",
		ArgsUsage: "<deployment-id|app-label>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  flagReveal,
				Usage: "Print the value of one output (e.g. access-auth-code)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			return handleMarketplaceOutputs(ctx, cmd.Args().First(), strings.TrimSpace(cmd.String(flagReveal)))
		},
	}
}

func handleMarketplaceOutputs(_ context.Context, reference, reveal string) error {
	deploymentID, err := resolveMarketplaceDeploymentID(reference)
	if err != nil {
		return err
	}
	if reveal != "" {
		value, downloadErr := api.DownloadMarketplaceDeploymentOutput(deploymentID, reveal)
		if downloadErr != nil {
			return utils.NewError(fmt.Sprintf("failed to reveal output %q: %s", reveal, downloadErr.Error()), nil)
		}
		// The value alone, so it can be piped or captured.
		fmt.Println(strings.TrimRight(string(value), "\n"))
		return nil
	}
	outputs, err := api.ListMarketplaceDeploymentOutputs(deploymentID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list outputs: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(outputs) {
		return nil
	}
	if len(outputs) == 0 {
		utils.PrintInfo("This app declares no outputs.")
		return nil
	}
	for _, output := range outputs {
		kind := "value"
		if output.Sensitive {
			kind = "sensitive"
		}
		utils.PrintStatusLine(output.Name, kind)
	}
	utils.PrintInfo("Reveal one with: 1ctl marketplace outputs %s --reveal <name>", reference)
	return nil
}

func resolveMarketplaceDeploymentID(reference string) (string, error) {
	if _, parseErr := uuid.Parse(reference); parseErr == nil {
		return reference, nil
	}
	namespace, err := satuskyctx.GetCurrentNamespaceOrError()
	if err != nil {
		return "", err
	}
	deployment, err := api.GetDeploymentByAppLabel(namespace, reference)
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("deployment %q not found: %s", reference, err.Error()), nil)
	}
	return deployment.DeploymentID.String(), nil
}
