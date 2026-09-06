// Package convex manages the platform-owned Convex service lifecycle.
package convex

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"1ctl/internal/api"
	"1ctl/internal/utils"
	"github.com/google/uuid"
	"github.com/urfave/cli/v3"
)

// Command returns the managed Convex command tree.
func Command() *cli.Command {
	return &cli.Command{Name: "convex", Usage: "Manage SatuSky managed Convex services", Commands: []*cli.Command{
		createCommand(),
		{Name: "list", Usage: "List Convex services in the selected organization", Action: func(_ context.Context, _ *cli.Command) error {
			items, err := api.ListConvex()
			if err != nil {
				return err
			}
			for i := range items {
				sanitize(&items[i])
			}
			if utils.PrintListOrJSON(items, "No Convex services found") {
				return nil
			}
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				rows = append(rows, []string{item.StorageID.String(), displayName(&item), item.StorageSize, item.StorageClass})
			}
			utils.PrintTable([]string{"ID", "NAME", "SIZE", "CLASS"}, rows)
			return nil
		}},
		instanceCommand("get", "Show configuration (without secrets)"),
		instanceCommand("status", "Observe workload readiness; does not verify public DNS/HTTPS"),
		instanceCommand("connection", "Show connection URLs without the instance secret"),
		instanceCommand("credentials", "Reveal the sensitive instance secret and connection URLs"),
		instanceCommand("redeploy", "Reconcile the existing service without changing its version"),
		instanceCommand("delete", "Delete service resources and database volume; retain Spaces bucket data"),
	}}
}

func createCommand() *cli.Command {
	var in api.ConvexCreateOptions
	return &cli.Command{Name: "create", Usage: "Provision Convex with isolated managed Spaces (asynchronous)", Flags: []cli.Flag{
		&cli.StringFlag{Name: "name", Required: true, Destination: &in.Name},
		&cli.StringFlag{Name: "size", Value: "10Gi", Destination: &in.StorageSize},
		&cli.StringFlag{Name: "storage-class", Usage: "Storage class (defaults to the cluster default)", Destination: &in.StorageClass},
		&cli.StringFlag{Name: "cpu-request", Value: "500m", Destination: &in.CPURequest},
		&cli.StringFlag{Name: "cpu", Value: "2", Destination: &in.CPULimit},
		&cli.StringFlag{Name: "memory-request", Value: "1Gi", Destination: &in.MemoryRequest},
		&cli.StringFlag{Name: "memory", Value: "2Gi", Destination: &in.MemoryLimit},
		&cli.BoolFlag{Name: "dashboard", Value: true, Destination: &in.DashboardEnabled},
	}, Action: func(_ context.Context, cmd *cli.Command) error {
		if cmd.Args().Len() != 0 {
			return fmt.Errorf("create takes flags, not positional arguments")
		}
		in.Name = strings.TrimSpace(in.Name)
		if err := validateCreate(in); err != nil {
			return err
		}
		if in.StorageClass == "" {
			classes, err := api.ListStorageClasses()
			if err != nil {
				return err
			}
			for _, class := range classes {
				if class.IsDefault {
					in.StorageClass = class.Name
					break
				}
			}
			if in.StorageClass == "" {
				return fmt.Errorf("no default storage class; provide --storage-class")
			}
		}
		item, err := api.CreateConvex(in)
		if err != nil {
			return err
		}
		sanitize(item)
		if utils.TryPrintJSON(item) {
			return nil
		}
		utils.PrintSuccess("Convex creation accepted (not yet ready)")
		printInstance(item)
		utils.PrintInfo("Check '1ctl convex status %s', then '1ctl convex connection %s'. Workload readiness does not verify public DNS/HTTPS.", item.StorageID, item.StorageID)
		return nil
	}}
}

var namePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
var quantityPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?(?:m|[KMGTPE]i?|[kMGTPE])?$`)

func validateCreate(in api.ConvexCreateOptions) error {
	if len(in.Name) > 63 || !namePattern.MatchString(in.Name) {
		return fmt.Errorf("name must be a lowercase DNS label, 1–63 characters")
	}
	for label, value := range map[string]string{"size": in.StorageSize, "cpu-request": in.CPURequest, "cpu": in.CPULimit, "memory-request": in.MemoryRequest, "memory": in.MemoryLimit} {
		if !quantityPattern.MatchString(value) || !strings.ContainsAny(value, "123456789") {
			return fmt.Errorf("%s must be a positive resource quantity", label)
		}
	}
	return nil
}

func instanceCommand(action, usage string) *cli.Command {
	cmd := &cli.Command{Name: action, Usage: usage, ArgsUsage: "<id-or-name>"}
	if action == "delete" {
		cmd.Aliases = []string{"destroy"}
		cmd.Flags = []cli.Flag{&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Confirm destructive service deletion"}}
	}
	cmd.Action = func(_ context.Context, cmd *cli.Command) error {
		if cmd.Args().Len() != 1 {
			return fmt.Errorf("convex %s requires exactly one service ID or name", action)
		}
		if action == "delete" && utils.IsJSONOutput() && !cmd.Bool("yes") {
			return fmt.Errorf("JSON deletion requires --yes")
		}
		item, err := resolve(cmd.Args().First())
		if err != nil {
			return err
		}
		id := item.StorageID.String()
		switch action {
		case "get":
			sanitize(item)
			if !utils.TryPrintJSON(item) {
				printInstance(item)
			}
		case "status":
			status, err := api.GetConvexStatus(id)
			if err != nil {
				return err
			}
			if !utils.TryPrintJSON(status) {
				utils.PrintStatusLine("Status", status.Status)
				utils.PrintStatusLine("Database ready", fmt.Sprint(status.DatabaseReady))
				utils.PrintStatusLine("Backend ready", fmt.Sprint(status.BackendReady))
				utils.PrintStatusLine("Dashboard enabled / ready", fmt.Sprintf("%t / %t", status.DashboardEnabled, status.DashboardReady))
				utils.PrintInfo("Public DNS/HTTPS reachability is not verified by workload status.")
			}
		case "connection", "credentials":
			connection, err := api.GetConvexCredentials(id)
			if err != nil {
				return err
			}
			if action == "connection" {
				connection.InstanceSecret = ""
			}
			if item.Convex != nil && !item.Convex.DashboardEnabled {
				connection.DashboardURL = ""
			}
			if !utils.TryPrintJSON(connection) {
				utils.PrintStatusLine("API URL", connection.APIURL)
				utils.PrintStatusLine("Site URL", connection.SiteURL)
				if connection.DashboardURL != "" {
					utils.PrintStatusLine("Dashboard URL", connection.DashboardURL)
				}
				utils.PrintStatusLine("TCP endpoint", connection.APIHost+":"+connection.APIPort)
				if action == "credentials" {
					utils.PrintStatusLine("Instance secret (sensitive)", connection.InstanceSecret)
				}
			}
		case "redeploy":
			if err := api.RedeployConvex(id); err != nil {
				return err
			}
			printAccepted(id, action)
		case "delete":
			if !utils.Confirm("Delete Convex "+displayName(item)+" and its database volume? Spaces bucket data is retained.", cmd.Bool("yes")) {
				return fmt.Errorf("deletion cancelled")
			}
			if err := api.DeleteConvex(id); err != nil {
				return err
			}
			printAccepted(id, action)
		}
		return nil
	}
	return cmd
}

func resolve(value string) (*api.StorageConfig, error) {
	if _, err := uuid.Parse(value); err == nil {
		return api.GetConvex(value)
	}
	items, err := api.ListConvex()
	if err != nil {
		return nil, err
	}
	var found *api.StorageConfig
	for _, item := range items {
		if displayName(&item) == value || (item.ClusterName != nil && *item.ClusterName == value) {
			if found != nil {
				return nil, fmt.Errorf("multiple Convex services named %q; use a storage ID", value)
			}
			copyItem := item
			found = &copyItem
		}
	}
	if found == nil {
		return nil, fmt.Errorf("convex service %q not found in the selected organization", value)
	}
	return found, nil
}

func displayName(item *api.StorageConfig) string {
	if item.Convex != nil && item.Convex.InstanceName != "" {
		return item.Convex.InstanceName
	}
	if item.ClusterName != nil {
		return *item.ClusterName
	}
	return item.StorageID.String()
}

// Legacy records may contain credential annotations. Never emit them through
// general-purpose get/list/create output, including JSON.
func sanitize(item *api.StorageConfig) { item.Annotations = nil }

func printInstance(item *api.StorageConfig) {
	utils.PrintStatusLine("ID", item.StorageID.String())
	utils.PrintStatusLine("Name", displayName(item))
	utils.PrintStatusLine("Storage", item.StorageSize+" ("+item.StorageClass+")")
}

func printAccepted(id, action string) {
	if !utils.TryPrintJSON(map[string]string{"storage_id": id, "action": action, "status": "accepted"}) {
		utils.PrintSuccess("Convex %s accepted; check service status for convergence", action)
	}
}
