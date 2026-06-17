package commands

import (
	"context"
	"1ctl/internal/api"
	"1ctl/internal/utils"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli/v3"
)

// ClusterCommand returns the cluster management command group.
func ClusterCommand() *cli.Command {
	return &cli.Command{
		Name:  "cluster",
		Usage: "View cluster and zone information",
		Commands: []*cli.Command{
			clusterZonesCommand(),
			clusterListCommand(),
		},
	}
}

func clusterZonesCommand() *cli.Command {
	return &cli.Command{
		Name:   "zones",
		Usage:  "List available deployment zones",
		Action: handleListZones,
	}
}

func clusterListCommand() *cli.Command {
	return &cli.Command{
		Name:   "list",
		Usage:  "List all enabled clusters",
		Action: handleListClusters,
	}
}

func handleListZones(ctx context.Context, cmd *cli.Command) error {
	zones, err := api.GetAvailableZones()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list zones: %s", err.Error()), nil)
	}

	if len(zones) == 0 {
		utils.PrintInfo("No zones available")
		return nil
	}

	if utils.TryPrintJSON(zones) {
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "ZONE\tLABEL\tCLUSTER") //nolint:errcheck
	_, _ = fmt.Fprintln(w, "----\t-----\t-------") //nolint:errcheck
	for _, z := range zones {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", z.Value, z.Label, z.ClusterID) //nolint:errcheck
	}
	_ = w.Flush() //nolint:errcheck

	return nil
}

func handleListClusters(ctx context.Context, cmd *cli.Command) error {
	clusters, err := api.GetClusters()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list clusters: %s", err.Error()), nil)
	}

	if len(clusters) == 0 {
		utils.PrintInfo("No clusters available")
		return nil
	}

	if utils.TryPrintJSON(clusters) {
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME\tZONE\tENDPOINT\tHEALTHY\tDEFAULT\tPRIORITY") //nolint:errcheck
	_, _ = fmt.Fprintln(w, "--\t----\t----\t--------\t-------\t-------\t--------") //nolint:errcheck
	for _, c := range clusters {
		healthStr := "✓"
		if !c.Healthy {
			healthStr = "✗"
		}
		defaultStr := ""
		if c.IsDefault {
			defaultStr = "★"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%d\n", //nolint:errcheck
			c.ID, c.Name, c.Zone, c.Endpoint, healthStr, defaultStr, c.Priority)
	}
	_ = w.Flush() //nolint:errcheck

	return nil
}
