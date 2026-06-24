// Package pricing defines the "1ctl pricing" command tree — flag names,
// input structs, and CLI wiring.  Handler logic lives in handlers.go.
package pricing

import (
	"context"
	"fmt"
	"os"
	"strings"

	"1ctl/internal/api"
	"1ctl/internal/utils"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------
// Single source of truth for flag names.  Used in Flag.Name (--help output)
// and nowhere else — the Destination struct fields are the canonical
// accessor.

const (
	flagConfigID     = "config-id"
	flagRegion       = "region"
	flagType         = "type"
	flagSLA          = "sla"
	flagMachineRefID = "machine-ref-id"
	flagMachineID    = "machine-id"
	flagStart        = "start"
	flagEnd          = "end"
)

// --- Allowed flag values ------------------------------------------------
// Used by Validator and ShellComplete for --type and --sla.

var (
	validMachineTypes = []string{"standard", "premium"}
	validSLATiers     = []string{"standard", "premium"}
	getAvailableZones = api.GetAvailableZones
)

const (
	completionCurrentEnv = "__1CTL_COMPLETE_CURRENT"
	completionPrevEnv    = "__1CTL_COMPLETE_PREV"
)

// --- Flag constructors --------------------------------------------------
// Wrappers around urfave/cli flag types.  Every constructor requires
// Destination — you cannot create a flag without wiring it to a struct
// field.  Validator is optional (nil = skip client-side validation).

func requiredString(name, usage string, dest *string, validate func(string) error) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		Destination: dest,
		Required:    true,
		Validator:   validate,
	}
}

func optionalString(name, usage string, dest *string, validate func(string) error) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		Destination: dest,
		Validator:   validate,
	}
}

// --- Input structs ------------------------------------------------------
// Each struct IS the binding target.  Flag Destinations write directly into
// struct fields — no per-flag variables, no constructor, no manual wiring.
//
// Adding a flag with Destination: &in.NewField without adding NewField to
// the struct is a compile error.

type pricingGetInput struct {
	ConfigID string
}

type pricingLookupInput struct {
	Region      string
	MachineType string
	SLA         string
}

type pricingCalculateInput struct {
	MachineRefID string
	MachineID    string
	StartTime    string
	EndTime      string
}

// --- Command tree -------------------------------------------------------
// Each factory scopes a single input struct var.  The framework populates it
// via Destination pointers.  The closure passes it straight to the handler.

// Command returns the root pricing command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:    "pricing",
		
		Usage:   "View machine pricing configurations",
		Commands: []*cli.Command{
			pricingListCommand(),
			pricingGetCommand(),
			pricingLookupCommand(),
			pricingCalculateCommand(),
		},
	}
}

func pricingListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all pricing configurations",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handlePricingList(ctx)
		},
	}
}

func pricingGetCommand() *cli.Command {
	var in pricingGetInput
	return &cli.Command{
		Name:  "get",
		Usage: "Get a pricing configuration by ID",
		Flags: []cli.Flag{
			requiredString(flagConfigID, "Pricing config ID", &in.ConfigID, nil),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handlePricingGet(ctx, in)
		},
	}
}

func pricingLookupCommand() *cli.Command {
	var in pricingLookupInput
	return &cli.Command{
		Name:  "lookup",
		Usage: "Look up pricing for a region, machine type, and SLA tier",
		Flags: []cli.Flag{
			requiredString(flagRegion, "Region (e.g. my-kul-1b)", &in.Region, nil),
			requiredString(flagType, "Machine type (e.g. standard, premium)", &in.MachineType, machineTypeValidator),
			requiredString(flagSLA, "SLA tier (e.g. standard, premium)", &in.SLA, slaTierValidator),
		},
		ShellComplete: pricingLookupShellComplete,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handlePricingLookup(ctx, in)
		},
	}
}

func pricingCalculateCommand() *cli.Command {
	var in pricingCalculateInput
	return &cli.Command{
		Name:  "calculate",
		Usage: "Calculate cost for a machine over a time range",
		Flags: []cli.Flag{
			requiredString(flagMachineRefID, "Machine reference ID (numeric)", &in.MachineRefID, utils.ValidateDigits),
			requiredString(flagMachineID, "Machine ID (UUID)", &in.MachineID, utils.ValidateUUID),
			requiredString(flagStart, "Start time (RFC3339 format)", &in.StartTime, utils.ValidateRFC3339),
			requiredString(flagEnd, "End time (RFC3339 format)", &in.EndTime, utils.ValidateRFC3339),
		},
		// Cross-field: start must be before end.  Per-flag Validator can't
		// check this — it only sees one value at a time.
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			return ctx, utils.ValidateTimeRange(in.StartTime, in.EndTime)
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handlePricingCalculate(ctx, in)
		},
	}
}

// --- Pricing-specific validators ----------------------------------------
// Thin wrappers around utils.ValidateEnum for --type and --sla.

func machineTypeValidator(s string) error {
	return utils.ValidateEnum(s, validMachineTypes)
}

func slaTierValidator(s string) error {
	return utils.ValidateEnum(s, validSLATiers)
}

func pricingLookupShellComplete(ctx context.Context, cmd *cli.Command) {
	current := os.Getenv(completionCurrentEnv)
	previous := os.Getenv(completionPrevEnv)
	if current == "" && previous == "" {
		previous = previousCompletionArg(os.Args)
	}

	if flag, ok := currentFlagAssignment(current); ok {
		switch flag {
		case flagType:
			printCompletionValues(cmd, validMachineTypes, "machine type", "--"+flagType+"=")
			return
		case flagSLA:
			printCompletionValues(cmd, validSLATiers, "SLA tier", "--"+flagSLA+"=")
			return
		case flagRegion:
			printRegionCompletionValues(cmd, "--"+flagRegion+"=")
			return
		}
	}

	switch currentFlagName(current) {
	case flagType:
		printCompletionValues(cmd, validMachineTypes, "machine type", "")
		return
	case flagSLA:
		printCompletionValues(cmd, validSLATiers, "SLA tier", "")
		return
	case flagRegion:
		printRegionCompletionValues(cmd, "")
		return
	}

	if strings.HasPrefix(current, "-") {
		cli.DefaultCompleteWithFlags(ctx, cmd)
		return
	}

	switch currentFlagName(previous) {
	case flagType:
		printCompletionValues(cmd, validMachineTypes, "machine type", "")
	case flagSLA:
		printCompletionValues(cmd, validSLATiers, "SLA tier", "")
	case flagRegion:
		printRegionCompletionValues(cmd, "")
		return
	default:
		cli.DefaultCompleteWithFlags(ctx, cmd)
	}
}

func previousCompletionArg(args []string) string {
	if len(args) < 2 || args[len(args)-1] != "--generate-shell-completion" {
		return ""
	}
	return args[len(args)-2]
}

func currentFlagAssignment(arg string) (flagName string, ok bool) {
	if !strings.HasPrefix(arg, "--") {
		return "", false
	}
	flag, _, found := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
	if !found {
		return "", false
	}
	return flag, true
}

func currentFlagName(arg string) string {
	arg = strings.TrimSpace(arg)
	if !strings.HasPrefix(arg, "--") || strings.Contains(arg, "=") {
		return ""
	}
	return strings.TrimPrefix(arg, "--")
}

func printCompletionValues(cmd *cli.Command, values []string, description string, prefix string) {
	for _, v := range values {
		fmt.Fprintf(cmd.Root().Writer, "%s%s:%s\n", prefix, v, description)
	}
}

func printRegionCompletionValues(cmd *cli.Command, prefix string) {
	zones, err := getAvailableZones()
	if err != nil || len(zones) == 0 {
		printCompletionValues(cmd, []string{"my-kul-1b", "my-bki-1a"}, "region", prefix)
		return
	}

	seen := make(map[string]bool, len(zones))
	for _, zone := range zones {
		if zone.Value == "" || seen[zone.Value] {
			continue
		}
		seen[zone.Value] = true

		description := zone.Label
		if description == "" {
			description = "region"
		}
		fmt.Fprintf(cmd.Root().Writer, "%s%s:%s\n", prefix, zone.Value, description)
	}
}
