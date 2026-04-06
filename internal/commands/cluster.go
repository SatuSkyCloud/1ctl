package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/utils"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli/v2"
)

// ClusterCommand returns the cluster management command group.
func ClusterCommand() *cli.Command {
	return &cli.Command{
		Name:  "cluster",
		Usage: "View cluster and zone information",
		Subcommands: []*cli.Command{
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

func handleListZones(_ *cli.Context) error {
	zones, err := api.GetAvailableZones()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list zones: %s", err.Error()), nil)
	}

	if len(zones) == 0 {
		utils.PrintInfo("No zones available")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ZONE\tLABEL\tCLUSTER")
	fmt.Fprintln(w, "----\t-----\t-------")
	for _, z := range zones {
		fmt.Fprintf(w, "%s\t%s\t%s\n", z.Value, z.Label, z.ClusterID)
	}
	w.Flush()

	return nil
}

func handleListClusters(_ *cli.Context) error {
	clusters, err := api.GetClusters()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list clusters: %s", err.Error()), nil)
	}

	if len(clusters) == 0 {
		utils.PrintInfo("No clusters available")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tZONE\tENDPOINT\tHEALTHY\tDEFAULT\tPRIORITY")
	fmt.Fprintln(w, "--\t----\t----\t--------\t-------\t-------\t--------")
	for _, c := range clusters {
		healthStr := "✓"
		if !c.Healthy {
			healthStr = "✗"
		}
		defaultStr := ""
		if c.IsDefault {
			defaultStr = "★"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%d\n",
			c.ID, c.Name, c.Zone, c.Endpoint, healthStr, defaultStr, c.Priority)
	}
	w.Flush()

	return nil
}
