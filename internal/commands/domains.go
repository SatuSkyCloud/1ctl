package commands

import (
	"fmt"
	"net"
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
			domainsSetupCommand(),
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
		Usage:     "Check backend, route, DNS, TLS, and HTTP status for a domain",
		ArgsUsage: "<domain>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "probe",
				Usage: "Run an HTTP reachability probe",
			},
		},
		Action: handleDomainsCheck,
	}
}

func domainsSetupCommand() *cli.Command {
	return &cli.Command{
		Name:      "setup",
		Usage:     "Show exact DNS setup instructions for a domain",
		ArgsUsage: "<domain>",
		Action:    handleDomainsSetup,
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
		if status, statusErr := api.GetDomainStatus(resp.IngressID.String(), domain, false); statusErr == nil {
			printDomainSetup(status)
		} else {
			utils.PrintInfo("Custom domain: run '1ctl domains setup %s' for exact DNS records.", domain)
		}
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
		return printDetachedDomainStatus(domain, err)
	}

	status, err := api.GetDomainStatus(ing.IngressID.String(), domain, c.Bool("probe"))
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to check domain %q: %s", domain, err.Error()), nil)
	}

	if utils.TryPrintJSON(status) {
		return nil
	}

	printDomainStatus(status)
	return nil
}

func handleDomainsSetup(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("domain is required. Usage: 1ctl domains setup <domain>", nil)
	}
	domain := c.Args().First()

	ing, err := api.GetIngressByDomainName(domain)
	if err != nil {
		return printDetachedDomainSetup(domain, err)
	}
	status, err := api.GetDomainStatus(ing.IngressID.String(), domain, false)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to load setup details for %q: %s", domain, err.Error()), nil)
	}
	if utils.TryPrintJSON(status) {
		return nil
	}

	printDomainSetup(status)
	return nil
}

func printDetachedDomainStatus(domain string, cause error) error {
	if utils.IsJSONOutput() {
		utils.TryPrintJSON(map[string]interface{}{
			"domain_name": domain,
			"attached":    false,
			"message":     cause.Error(),
		})
		return nil
	}
	utils.PrintHeader("Domain %s", domain)
	utils.PrintStatusLine("Backend", "not attached")
	utils.PrintStatusLine("Route", "not attached")
	utils.PrintStatusLine("DNS", "not checked")
	utils.PrintStatusLine("TLS", "unknown")
	utils.PrintStatusLine("HTTP", "not checked")
	utils.PrintInfo("Attach it with: 1ctl domains add %s --app <app>", domain)
	return nil
}

func printDetachedDomainSetup(domain string, cause error) error {
	if utils.IsJSONOutput() {
		utils.TryPrintJSON(map[string]interface{}{
			"domain_name": domain,
			"attached":    false,
			"message":     cause.Error(),
		})
		return nil
	}
	utils.PrintHeader("Domain Setup %s", domain)
	utils.PrintStatusLine("Backend", "not attached")
	utils.PrintInfo("Attach it first with: 1ctl domains add %s --app <app>", domain)
	return nil
}

func printDomainStatus(status *api.DomainStatusResponse) {
	utils.PrintHeader("Domain %s", status.DomainName)
	utils.PrintStatusLine("Backend", fmt.Sprintf("attached to %s in %s", status.AppLabel, status.Namespace))
	utils.PrintStatusLine("Route", domainRouteText(status.Route))
	utils.PrintStatusLine("DNS", domainDNSText(status.DNS))
	utils.PrintStatusLine("TLS", domainTLSText(status.TLS))
	utils.PrintStatusLine("HTTP", domainHTTPText(status.Reachability))
	if status.DNS.Status != api.DNSStatusResolved {
		utils.PrintInfo("Run setup details with: 1ctl domains setup %s", status.DomainName)
	}
}

func printDomainSetup(status *api.DomainStatusResponse) {
	utils.PrintHeader("Domain Setup %s", status.DomainName)
	utils.PrintStatusLine("App", status.AppLabel)
	utils.PrintStatusLine("Namespace", status.Namespace)
	utils.PrintStatusLine("Current DNS", domainDNSText(status.DNS))
	utils.PrintStatusLine("TLS", domainTLSText(status.TLS))

	if status.DNS.ExpectedIP == "" {
		utils.PrintWarning("Backend did not return an expected DNS target yet.")
		return
	}

	recordType := "A"
	if net.ParseIP(status.DNS.ExpectedIP) == nil {
		recordType = "CNAME"
	}
	utils.PrintHeader("Required DNS Records")
	utils.PrintStatusLine("Type", recordType)
	utils.PrintStatusLine("Name", status.DomainName)
	utils.PrintStatusLine("Value", status.DNS.ExpectedIP)
	utils.PrintInfo("Next check: 1ctl domains check %s --probe", status.DomainName)
}

func domainRouteText(status api.DomainRouteStatus) string {
	if !status.Attached {
		if status.Message != "" {
			return "not attached: " + status.Message
		}
		return "not attached"
	}
	if status.ResourceKind == "" && status.ResourceName == "" {
		return "attached"
	}
	return fmt.Sprintf("attached to %s/%s", status.ResourceKind, status.ResourceName)
}

func domainDNSText(status api.DNSStatusResponse) string {
	parts := []string{string(status.Status)}
	if len(status.ResolvedIPs) > 0 {
		parts = append(parts, "resolved "+strings.Join(status.ResolvedIPs, ", "))
	}
	if status.ExpectedIP != "" {
		parts = append(parts, "expected "+status.ExpectedIP)
	}
	if status.Message != "" {
		parts = append(parts, status.Message)
	}
	return strings.Join(parts, " - ")
}

func domainTLSText(status api.TLSStatusResponse) string {
	if status.Status == "" {
		return "unknown"
	}
	if status.ExpiresAt != nil {
		return fmt.Sprintf("%s, expires %s", status.Status, status.ExpiresAt.Format("2006-01-02"))
	}
	if status.Message != "" {
		return fmt.Sprintf("%s - %s", status.Status, status.Message)
	}
	return string(status.Status)
}

func domainHTTPText(status api.DomainReachabilityStatus) string {
	if !status.Checked {
		if status.Message != "" {
			return "not checked - " + status.Message
		}
		return "not checked"
	}
	if status.Reachable {
		return fmt.Sprintf("reachable %s %d", status.URL, status.StatusCode)
	}
	if status.Message != "" {
		return "not reachable - " + status.Message
	}
	return "not reachable"
}
