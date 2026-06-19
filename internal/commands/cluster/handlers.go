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
	_, _ = fmt.Fprintln(w, "ZONE\tLABEL\tCLUSTER")
	_, _ = fmt.Fprintln(w, "----\t-----\t-------")
	for _, z := range zones {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", z.Value, z.Label, z.ClusterID)
	}
	_ = w.Flush()

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
	_, _ = fmt.Fprintln(w, "ID\tNAME\tZONE\tENDPOINT\tHEALTHY\tDEFAULT\tPRIORITY")
	_, _ = fmt.Fprintln(w, "--\t----\t----\t--------\t-------\t-------\t--------")
	for _, c := range clusters {
		healthStr := "✓"
		if !c.Healthy {
			healthStr = "✗"
		}
		defaultStr := ""
		if c.IsDefault {
			defaultStr = "★"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%d\n",
			c.ID, c.Name, c.Zone, c.Endpoint, healthStr, defaultStr, c.Priority)
	}
	_ = w.Flush()

	return nil
}
