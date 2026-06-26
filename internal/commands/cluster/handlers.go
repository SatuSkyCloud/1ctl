package cluster

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"1ctl/internal/api"
	"1ctl/internal/utils"
)

// --- Handlers -----------------------------------------------------------

func handleListZones(ctx context.Context) error {
	zones, err := api.GetAvailableZones()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list zones: %s", err.Error()), nil)
	}

	if utils.PrintListOrJSON(zones, "No zones available") {
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if _, err := fmt.Fprintln(w, "ZONE\tLABEL\tCLUSTER"); err != nil {
		return utils.NewError(fmt.Sprintf("failed to write zone table header: %s", err.Error()), nil)
	}
	if _, err := fmt.Fprintln(w, "----\t-----\t-------"); err != nil {
		return utils.NewError(fmt.Sprintf("failed to write zone table header: %s", err.Error()), nil)
	}
	for _, z := range zones {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", z.Value, z.Label, z.ClusterID); err != nil {
			return utils.NewError(fmt.Sprintf("failed to write zone row: %s", err.Error()), nil)
		}
	}
	if err := w.Flush(); err != nil {
		return utils.NewError(fmt.Sprintf("failed to flush zone table: %s", err.Error()), nil)
	}

	return nil
}

func handleListClusters(ctx context.Context) error {
	clusters, err := api.GetClusters()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list clusters: %s", err.Error()), nil)
	}

	if utils.PrintListOrJSON(clusters, "No clusters available") {
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if _, err := fmt.Fprintln(w, "ID\tNAME\tZONE\tENDPOINT\tHEALTHY\tDEFAULT\tPRIORITY"); err != nil {
		return utils.NewError(fmt.Sprintf("failed to write cluster table header: %s", err.Error()), nil)
	}
	if _, err := fmt.Fprintln(w, "--\t----\t----\t--------\t-------\t-------\t--------"); err != nil {
		return utils.NewError(fmt.Sprintf("failed to write cluster table header: %s", err.Error()), nil)
	}
	for _, c := range clusters {
		healthStr := "✓"
		if !c.Healthy {
			healthStr = "✗"
		}
		defaultStr := ""
		if c.IsDefault {
			defaultStr = "★"
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%d\n",
			c.ID, c.Name, c.Zone, c.Endpoint, healthStr, defaultStr, c.Priority); err != nil {
			return utils.NewError(fmt.Sprintf("failed to write cluster row: %s", err.Error()), nil)
		}
	}
	if err := w.Flush(); err != nil {
		return utils.NewError(fmt.Sprintf("failed to flush cluster table: %s", err.Error()), nil)
	}

	return nil
}
