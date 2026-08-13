package deploy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"1ctl/internal/api"
	deploypkg "1ctl/internal/deploy"
	"1ctl/internal/utils"

	"github.com/urfave/cli/v3"
)

// EventsInput holds flags for the "events" subcommand.
type EventsInput struct {
	DeploymentID string
	App          string
	Config       string
	Follow       bool
	Since        int
}

func appEventsCommand() *cli.Command {
	var in EventsInput
	return &cli.Command{
		Name:      "events",
		Usage:     "Show what has happened to an application",
		ArgsUsage: "<app-name>",
		Description: `Show the lifecycle timeline for an application: when it was accepted,
   when each reconciliation ran, when its hostname was reserved, and what DNS
   has actually observed.

   Use --follow to keep watching while a deployment settles:
      1ctl app events my-app --follow`,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "deployment-id", Usage: "Deployment ID (alternative to positional arg)", Destination: &in.DeploymentID},
			&cli.StringFlag{Name: flagConfig, Usage: "Config name or path", Destination: &in.Config},
			&cli.BoolFlag{Name: "follow", Aliases: []string{"f"}, Usage: "Keep watching for new events", Destination: &in.Follow},
			&cli.IntFlag{Name: "last", Usage: "Show only the most recent N events (0 shows all)", Destination: &in.Since},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() >= 1 {
				arg := cmd.Args().First()
				if looksLikeUUID(arg) {
					in.DeploymentID = arg
				} else {
					in.App = arg
				}
			}
			return handleAppEvents(ctx, in)
		},
	}
}

const (
	eventsFollowInterval = 3 * time.Second
	// eventsFollowLimit bounds an unattended --follow so a forgotten terminal
	// does not poll the API indefinitely.
	eventsFollowLimit = 15 * time.Minute
)

func handleAppEvents(ctx context.Context, in EventsInput) error {
	deploymentID, err := deploypkg.ResolveDeploymentID(in.DeploymentID, in.App, in.Config)
	if err != nil {
		return err
	}

	timeline, err := api.GetDeploymentEvents(deploymentID)
	if err != nil {
		return err
	}
	if utils.TryPrintJSON(timeline) {
		return nil
	}

	events := trimToLast(timeline.Events, in.Since)
	if len(events) == 0 {
		fmt.Printf("No events recorded yet for %s\n", timeline.AppLabel)
		if !in.Follow {
			return nil
		}
	}

	utils.PrintHeader("Events for %s", timeline.AppLabel)
	seen := make(map[string]struct{}, len(events))
	for _, event := range events {
		printEvent(event)
		seen[event.Key()] = struct{}{}
	}
	// Events already shown above the --last cut-off must still be suppressed, or
	// --follow would reprint the whole history on its first poll.
	for _, event := range timeline.Events {
		seen[event.Key()] = struct{}{}
	}
	printCurrentState(deploymentID)
	if !in.Follow {
		return nil
	}
	return followEvents(ctx, deploymentID, seen)
}

// printCurrentState answers the question the timeline itself cannot: whether the
// app is up right now.
//
// It is deliberately not an event. Readiness is live Kubernetes state with no
// durable "became ready at" timestamp, and inventing one would put a fabricated
// entry in a timeline whose whole value is that every line is a recorded fact.
func printCurrentState(deploymentID string) {
	status, err := api.GetLiveDeploymentStatus(deploymentID)
	if err != nil || status == nil || status.Readiness == nil {
		return
	}
	readiness := status.Readiness
	// Only conditions the server actually reported are shown. Rendering an
	// absent condition as "unknown" adds a word and no information, and reads as
	// a problem where there is none.
	parts := make([]string, 0, 3)
	for _, condition := range []struct{ label, state string }{
		{"workload", readiness.Workload.State},
		{"application", readiness.Application.State},
		{"dns", readiness.DNS.State},
	} {
		state := strings.TrimSpace(condition.state)
		if state == "" || strings.EqualFold(state, "not_applicable") {
			continue
		}
		parts = append(parts, condition.label+" "+state)
	}
	if len(parts) == 0 {
		return
	}
	summary := "Current: " + strings.Join(parts, " · ")

	if strings.EqualFold(readiness.Application.State, "verified") &&
		strings.EqualFold(readiness.Workload.State, "complete") {
		utils.PrintSuccess("%s", summary)
		return
	}
	utils.PrintInfo("%s", summary)
}

func followEvents(ctx context.Context, deploymentID string, seen map[string]struct{}) error {
	deadline := time.Now().Add(eventsFollowLimit)
	ticker := time.NewTicker(eventsFollowInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
		timeline, err := api.GetDeploymentEvents(deploymentID)
		if err != nil {
			// A transient API failure must not end the watch; the next tick
			// retries and the user keeps their stream.
			utils.PrintWarning("Could not refresh events: %s", err.Error())
			continue
		}
		for _, event := range timeline.Events {
			if _, already := seen[event.Key()]; already {
				continue
			}
			printEvent(event)
			seen[event.Key()] = struct{}{}
		}
		if time.Now().After(deadline) {
			utils.PrintInfo("Stopped following after %s. Run the command again to continue.", eventsFollowLimit)
			return nil
		}
	}
}

func trimToLast(events []api.DeploymentEvent, last int) []api.DeploymentEvent {
	if last <= 0 || len(events) <= last {
		return events
	}
	return events[len(events)-last:]
}

func printEvent(event api.DeploymentEvent) {
	label := event.Category
	if event.Type != "" {
		label = event.Category + "/" + event.Type
	}
	line := fmt.Sprintf("%s  %-26s %s",
		event.At.Local().Format("15:04:05"), label, event.Message)
	if detail := formatEventDetail(event); detail != "" {
		line += "  " + detail
	}
	switch event.Level {
	case "error":
		utils.PrintError("%s", line)
	case "warning":
		utils.PrintWarning("%s", line)
	default:
		fmt.Println(line)
	}
}

// formatEventDetail renders only the fields a reader acts on. Echoing the whole
// detail map would bury the error behind the expected target on every line.
func formatEventDetail(event api.DeploymentEvent) string {
	parts := make([]string, 0, 3)
	for _, key := range []string{"error", "code", "fqdn", "image"} {
		if value := strings.TrimSpace(event.Detail[key]); value != "" {
			parts = append(parts, fmt.Sprintf("%s=%s", key, value))
			// One qualifier is enough per line; the JSON output carries the rest.
			break
		}
	}
	if event.Generation != nil {
		parts = append(parts, fmt.Sprintf("gen=%d", *event.Generation))
	}
	if len(parts) == 0 {
		return ""
	}
	return "(" + strings.Join(parts, " ") + ")"
}
