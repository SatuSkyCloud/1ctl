package pricing

import (
	"bytes"
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"1ctl/internal/api"

	"github.com/urfave/cli/v3"
)

// TestFlagsHaveDestination ensures every Required flag in the pricing
// command tree has a Destination pointer.  A Required flag without one
// demands user input but discards the value — the handler never sees it.
func TestFlagsHaveDestination(t *testing.T) {
	walkCommands(Command(), func(cmd *cli.Command) {
		for _, f := range cmd.Flags {
			if !isRequired(f) {
				continue
			}
			if hasNilDestination(f) {
				t.Errorf("command %q: required flag %q has no Destination — value will be lost", cmd.Name, flagName(f))
			}
		}
	})
}

func TestPricingLookupShellCompleteFlagNames(t *testing.T) {
	out := runPricingLookupCompletion(t, "--", "lookup", "--")

	assertContains(t, out, "--region:Region (e.g. my-kul-1b)")
	assertContains(t, out, "--type:Machine type (e.g. standard, premium)")
	assertContains(t, out, "--sla:SLA tier (e.g. standard, premium)")
	assertNotContains(t, out, "standard:")
	assertNotContains(t, out, "premium:")
}

func TestPricingLookupShellCompleteTypeValues(t *testing.T) {
	out := runPricingLookupCompletion(t, "", "--type", "--type")

	if out != "standard:machine type\npremium:machine type\n" {
		t.Fatalf("completion output = %q", out)
	}
}

func TestPricingLookupShellCompleteSLAValues(t *testing.T) {
	out := runPricingLookupCompletion(t, "", "--sla", "--sla")

	if out != "standard:SLA tier\npremium:SLA tier\n" {
		t.Fatalf("completion output = %q", out)
	}
}

func TestPricingLookupShellCompleteRegionFallbackValues(t *testing.T) {
	stubAvailableZones(t, nil, errors.New("offline"))

	out := runPricingLookupCompletion(t, "", "--region", "--region")

	if out != "my-kul-1b:region\nmy-bki-1a:region\n" {
		t.Fatalf("completion output = %q", out)
	}
}

func TestPricingLookupShellCompleteRegionAPIValues(t *testing.T) {
	stubAvailableZones(t, []api.ZoneOption{
		{Value: "my-kul-1b", Label: "Kuala Lumpur 1B"},
		{Value: "my-bki-1a", Label: "Kota Kinabalu 1A"},
		{Value: "my-kul-1b", Label: "duplicate"},
	}, nil)

	out := runPricingLookupCompletion(t, "", "--region", "--region")

	if out != "my-kul-1b:Kuala Lumpur 1B\nmy-bki-1a:Kota Kinabalu 1A\n" {
		t.Fatalf("completion output = %q", out)
	}
}

func TestPricingLookupShellCompleteEqualsForm(t *testing.T) {
	out := runPricingLookupCompletion(t, "--sla=p", "lookup", "--sla=p")

	if out != "--sla=standard:SLA tier\n--sla=premium:SLA tier\n" {
		t.Fatalf("completion output = %q", out)
	}
}

func runPricingLookupCompletion(t *testing.T, current, previous string, args ...string) string {
	t.Helper()
	t.Setenv(completionCurrentEnv, current)
	t.Setenv(completionPrevEnv, previous)

	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })

	out := &bytes.Buffer{}
	cmd := Command()
	cmd.EnableShellCompletion = true
	cmd.Writer = out

	runArgs := append([]string{"pricing", "lookup"}, args...)
	runArgs = append(runArgs, "--generate-shell-completion")
	os.Args = runArgs
	if err := cmd.Run(context.Background(), runArgs); err != nil {
		t.Fatalf("completion run failed: %v", err)
	}
	return out.String()
}

func stubAvailableZones(t *testing.T, zones []api.ZoneOption, err error) {
	t.Helper()
	original := getAvailableZones
	getAvailableZones = func() ([]api.ZoneOption, error) {
		return zones, err
	}
	t.Cleanup(func() {
		getAvailableZones = original
	})
}

func assertContains(t *testing.T, s, want string) {
	t.Helper()
	if !strings.Contains(s, want) {
		t.Fatalf("expected %q to contain %q", s, want)
	}
}

func assertNotContains(t *testing.T, s, notWant string) {
	t.Helper()
	if strings.Contains(s, notWant) {
		t.Fatalf("expected %q not to contain %q", s, notWant)
	}
}

func walkCommands(cmd *cli.Command, fn func(*cli.Command)) {
	fn(cmd)
	for _, sub := range cmd.Commands {
		walkCommands(sub, fn)
	}
}

// isRequired checks whether the flag's Required field is true.
// urfave/cli v3 embeds FlagBase in every concrete flag type, so we
// can reach Required via reflection without a type switch.
func isRequired(f cli.Flag) bool {
	return reflect.ValueOf(f).Elem().FieldByName("Required").Bool()
}

// hasNilDestination checks whether the flag's Destination pointer is nil.
func hasNilDestination(f cli.Flag) bool {
	dest := reflect.ValueOf(f).Elem().FieldByName("Destination")
	if !dest.IsValid() {
		return true // missing field = no destination
	}
	return dest.IsNil()
}

// flagName returns the Name field via reflection.
func flagName(f cli.Flag) string {
	return reflect.ValueOf(f).Elem().FieldByName("Name").String()
}
