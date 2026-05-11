package commands

import (
	"fmt"
	"strings"

	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/utils"

	"github.com/urfave/cli/v2"
)

// DomainsCommand exposes app-oriented domain/TLS workflows over the same
// backend routes that `ingress` uses, but resolves deployment, service,
// namespace, and ingress IDs internally from the user's app name. Users
// don't see Kubernetes Ingress or cert-manager Issuer concepts.
func DomainsCommand() *cli.Command {
	return &cli.Command{
		Name:    "domains",
		Aliases: []string{"domain"},
		Usage:   "Add, list, and inspect custom domains for your apps",
		Subcommands: []*cli.Command{
			domainsListCommand(),
			domainsAddCommand(),
			domainsRemoveCommand(),
			domainsCheckCommand(),
		},
	}
}

func domainsListCommand() *cli.Command {
	return &cli.Command{
		Name:   "list",
		Usage:  "List all domains in the current organization",
		Action: handleDomainsList,
	}
}

func domainsAddCommand() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     "Add a custom domain to an app",
		ArgsUsage: "<domain>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "app",
				Usage:    "App name (the value of [app] name in satusky.toml, or --name on deploy)",
				Required: true,
			},
			&cli.IntFlag{
				Name:  "port",
				Usage: "Target port on the deployment",
				Value: 8080,
			},
			&cli.BoolFlag{
				Name:  "custom-dns",
				Usage: "Use Let's Encrypt for TLS (required for non-*.satusky.com hostnames)",
			},
		},
		Action: handleDomainsAdd,
	}
}

func domainsRemoveCommand() *cli.Command {
	return &cli.Command{
		Name:      "remove",
		Aliases:   []string{"rm"},
		Usage:     "Remove a custom domain from an app",
		ArgsUsage: "<domain>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "app",
				Usage:    "App name (the value of [app] name in satusky.toml)",
				Required: true,
			},
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Usage:   "Skip confirmation prompt",
			},
		},
		Action: handleDomainsRemove,
	}
}

func domainsCheckCommand() *cli.Command {
	return &cli.Command{
		Name:      "check",
		Usage:     "Check DNS / TLS status for a domain",
		ArgsUsage: "<domain>",
		Action:    handleDomainsCheck,
	}
}

func handleDomainsList(c *cli.Context) error {
	if _, err := context.GetCurrentNamespaceOrError(); err != nil {
		return err
	}
	ingresses, err := api.ListIngresses()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list domains: %s", err.Error()), nil)
	}

	if len(ingresses) == 0 {
		utils.PrintInfo("No domains yet. Add one with: 1ctl domains add <domain> --app <app>")
		return nil
	}

	if utils.TryPrintJSON(ingresses) {
		return nil
	}

	headers := []string{"DOMAIN", "APP", "TLS", "CREATED"}
	rows := make([][]string, 0, len(ingresses))
	for _, ing := range ingresses {
		tls := "platform"
		if ing.DnsConfig == api.DnsConfigCustom {
			tls = "letsencrypt"
		}
		rows = append(rows, []string{
			ing.DomainName,
			ing.AppLabel,
			tls,
			api.FormatTimeAgo(ing.CreatedAt),
		})
	}
	utils.PrintTable(headers, rows)
	return nil
}

func handleDomainsAdd(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("domain is required. Usage: 1ctl domains add <domain> --app <app>", nil)
	}
	domain := c.Args().First()
	appName := c.String("app")

	namespace, err := context.GetCurrentNamespaceOrError()
	if err != nil {
		return err
	}

	// Resolve the deployment by app label so the user doesn't supply UUIDs.
	dep, err := api.GetDeploymentByAppLabel(namespace, appName)
	if err != nil {
		return utils.NewError(fmt.Sprintf("could not find an app named %q in the current organization: %s", appName, err.Error()), nil)
	}

	// Find the matching Service for this deployment so we can wire the ingress.
	services, err := api.ListServices()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list services: %s", err.Error()), nil)
	}
	var serviceID string
	for _, s := range services {
		if s.DeploymentID == dep.DeploymentID {
			serviceID = s.ServiceID.String()
			break
		}
	}
	if serviceID == "" {
		return utils.NewError(fmt.Sprintf("no service found for app %q — has the app been deployed?", appName), nil)
	}

	port, err := api.SafeInt32(c.Int("port"))
	if err != nil {
		return utils.NewError("invalid --port value", err)
	}

	// Auto-pick TLS strategy: custom hostnames go through Let's Encrypt unless
	// the user explicitly opts out; *.satusky.com hostnames use the platform.
	dnsCfg := api.DnsConfigDefault
	isSatuskyHost := strings.HasSuffix(strings.ToLower(domain), ".satusky.com") || strings.ToLower(domain) == "satusky.com"
	if c.Bool("custom-dns") || !isSatuskyHost {
		dnsCfg = api.DnsConfigCustom
	}

	ingress := api.Ingress{
		DeploymentID: dep.DeploymentID,
		ServiceID:    api.ToUUID(serviceID),
		AppLabel:     appName,
		Namespace:    namespace,
		DomainName:   domain,
		DnsConfig:    dnsCfg,
		Port:         port,
	}
	resp, err := api.UpsertIngress(ingress)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to add domain: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Domain %s attached to app %s", resp.DomainName, appName)
	if dnsCfg == api.DnsConfigCustom {
		utils.PrintInfo("Custom domain: point an A record at the platform LoadBalancer IP, then run:")
		utils.PrintInfo("  1ctl domains check %s", domain)
	}
	return nil
}

func handleDomainsRemove(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("domain is required. Usage: 1ctl domains remove <domain> --app <app>", nil)
	}
	domain := c.Args().First()
	appName := c.String("app")

	if _, err := context.GetCurrentNamespaceOrError(); err != nil {
		return err
	}

	ing, err := api.GetIngressByDomainName(domain)
	if err != nil {
		return utils.NewError(fmt.Sprintf("no domain %q found in this organization: %s", domain, err.Error()), nil)
	}
	if ing.AppLabel != appName {
		return utils.NewError(fmt.Sprintf("domain %q belongs to app %q, not %q — refusing to remove without explicit match", domain, ing.AppLabel, appName), nil)
	}
	if !utils.Confirm(fmt.Sprintf("Remove domain %s from app %s?", domain, appName), c.Bool("yes")) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeleteIngress(ing.IngressID.String()); err != nil {
		return utils.NewError(fmt.Sprintf("failed to remove domain: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Domain %s removed from app %s", domain, appName)
	return nil
}

func handleDomainsCheck(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("domain is required. Usage: 1ctl domains check <domain>", nil)
	}
	domain := c.Args().First()

	ing, err := api.GetIngressByDomainName(domain)
	if err != nil {
		return utils.NewError(fmt.Sprintf("no domain %q registered: %s", domain, err.Error()), nil)
	}

	if utils.TryPrintJSON(ing) {
		return nil
	}

	utils.PrintHeader("Domain %s", domain)
	utils.PrintStatusLine("App", ing.AppLabel)
	utils.PrintStatusLine("Namespace", ing.Namespace)
	if ing.DnsConfig == api.DnsConfigCustom {
		utils.PrintStatusLine("TLS", "Let's Encrypt (custom domain)")
		utils.PrintStatusLine("DNS requirement", "A record must point at platform LoadBalancer IP")
	} else {
		utils.PrintStatusLine("TLS", "Platform-managed (*.satusky.com)")
	}
	utils.PrintStatusLine("Created", api.FormatTimeAgo(ing.CreatedAt))
	return nil
}
