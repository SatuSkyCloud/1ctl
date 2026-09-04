package satusky

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Snapshot is a bounded read-only view of the user's SatuSky state,
// gathered by running real 1ctl commands. Failures are recorded per
// command in Errors instead of aborting the batch; when unauthenticated
// the batch stops after auth status so there is no failed-command noise.
type Snapshot struct {
	Authenticated bool
	Profile       string
	Org           string
	Namespace     string
	Apps          []string
	Postgres      []string
	Valkey        []string
	NATS          []string
	Domains       []string
	Credits       string
	Errors        []string
	TakenAt       time.Time
}

// maxListEntries caps each name list in the digest; longer lists render
// as "…N more".
const maxListEntries = 20

// snapshotSteps are the read-only commands (canonical spellings verified
// against contracts/cli.json) that follow a successful auth status check.
type snapshotStep struct {
	args  []string
	apply func(s *Snapshot, res Result)
}

var snapshotSteps = []snapshotStep{
	{[]string{"profile", "list"}, applyProfileList},
	{[]string{"app", "list"}, applyAppsList},
	{[]string{"postgres", "list"}, applyPostgresList},
	{[]string{"valkey", "list"}, applyValkeyList},
	{[]string{"nats", "list"}, applyNATSList},
	{[]string{"domains", "list"}, applyDomainsList},
	{[]string{"credits", "balance"}, applyCreditsBalance},
}

// Snapshot gathers the SatuSky state via a bounded batch of read-only
// 1ctl commands. auth status gates the batch: a non-zero exit means the
// user is not authenticated and the remaining (API-bound) commands are
// skipped. Each later failure is recorded as an Errors entry.
func (r *Runner) Snapshot(ctx context.Context) (*Snapshot, error) {
	snap := &Snapshot{TakenAt: time.Now()}
	authRes, err := r.Run(ctx, "auth", "status")
	if err != nil {
		return nil, fmt.Errorf("run 1ctl auth status: %w", err)
	}
	if authRes.ExitCode != 0 {
		snap.Errors = append(snap.Errors, "auth status")
		return snap, nil // not authenticated; skip the rest cheaply
	}
	snap.Authenticated = true
	parseAuthStatus(snap, authRes.Stdout)

	for _, step := range snapshotSteps {
		res, err := r.RunJSON(ctx, step.args...)
		if err != nil || res.ExitCode != 0 {
			snap.Errors = append(snap.Errors, strings.Join(step.args, " "))
			continue
		}
		step.apply(snap, res)
	}
	return snap, nil
}

// Digest renders the snapshot as a compact block for the system prompt
// (and /status). It stays under ~1.5k characters: counts + names per
// resource, capped at maxListEntries, plus a short errors line.
func (s *Snapshot) Digest() string {
	var b strings.Builder
	b.WriteString("## Current SatuSky state\n")
	if !s.Authenticated {
		b.WriteString("not authenticated — SatuSky actions unavailable; suggest `1ctl auth login`\n")
		return b.String()
	}
	b.WriteString("authenticated: yes")
	if s.Profile != "" {
		b.WriteString(" · profile: " + s.Profile)
	}
	if s.Org != "" {
		b.WriteString(" · org: " + s.Org)
	}
	if s.Namespace != "" {
		b.WriteString(" · namespace: " + s.Namespace)
	}
	b.WriteString("\n")
	b.WriteString("apps: " + joinNames(s.Apps) + "\n")
	b.WriteString("postgres: " + joinNames(s.Postgres) + "\n")
	b.WriteString("valkey: " + joinNames(s.Valkey) + "\n")
	b.WriteString("nats: " + joinNames(s.NATS) + "\n")
	b.WriteString("domains: " + joinNames(s.Domains) + "\n")
	if s.Credits != "" {
		b.WriteString("credits: " + s.Credits + "\n")
	}
	if len(s.Errors) > 0 {
		fmt.Fprintf(&b, "(%d state checks failed: %s)\n", len(s.Errors), strings.Join(s.Errors, ", "))
	}
	return b.String()
}

// joinNames renders a name list: "a, b, c …2 more" (capped at
// maxListEntries), or "none" when empty.
func joinNames(names []string) string {
	if len(names) == 0 {
		return "none"
	}
	shown := names
	if len(shown) > maxListEntries {
		shown = shown[:maxListEntries]
	}
	line := strings.Join(shown, ", ")
	if len(names) > maxListEntries {
		line += fmt.Sprintf(" …%d more", len(names)-maxListEntries)
	}
	return line
}

// parseAuthStatus extracts org and namespace from `1ctl auth status`
// text output (that command prints text even with -o json).
func parseAuthStatus(s *Snapshot, stdout string) {
	for _, line := range strings.Split(stdout, "\n") {
		key, val, ok := splitStatusLine(line)
		if !ok {
			continue
		}
		switch key {
		case "Organization":
			s.Org = val
		case "Namespace":
			s.Namespace = val
		}
	}
}

// splitStatusLine splits a "Key: value" status line.
func splitStatusLine(line string) (key, val string, ok bool) {
	line = strings.TrimSpace(line)
	i := strings.Index(line, ":")
	if i <= 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:i])
	val = strings.TrimSpace(line[i+1:])
	if key == "" || val == "" {
		return "", "", false
	}
	return key, val, true
}

// applyProfileList picks the active profile from `1ctl profile list`
// JSON ([{name, api_url, email, active}]); text fallback scans for the
// "* " active marker.
func applyProfileList(s *Snapshot, res Result) {
	if s.Profile = profileFromJSON(res.Stdout); s.Profile != "" {
		return
	}
	s.Profile = profileFromText(res.Stdout)
}

func profileFromJSON(stdout string) string {
	var profiles []struct {
		Name     string `json:"name"`
		IsActive bool   `json:"active"`
	}
	if json.Unmarshal([]byte(stdout), &profiles) != nil {
		return ""
	}
	for _, p := range profiles {
		if p.IsActive && p.Name != "" {
			return p.Name
		}
	}
	return ""
}

func profileFromText(stdout string) string {
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "* ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "* "))
		}
	}
	return ""
}

// nameFromJSON parses a JSON array of objects and extracts the first
// non-empty value for any of the given keys (used for app_label /
// cluster_name / domain_name).
func nameFromJSON(stdout string, keys ...string) []string {
	var items []map[string]any
	if json.Unmarshal([]byte(stdout), &items) != nil {
		return nil
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		for _, key := range keys {
			if v, ok := item[key].(string); ok && v != "" {
				names = append(names, v)
				break
			}
		}
	}
	return names
}

func applyAppsList(s *Snapshot, res Result) {
	if names := nameFromJSON(res.Stdout, "app_label", "name"); names != nil {
		s.Apps = names
		return
	}
	s.Apps = namesFromText(res.Stdout)
}

func applyPostgresList(s *Snapshot, res Result) {
	if names := nameFromJSON(res.Stdout, "cluster_name", "name"); names != nil {
		s.Postgres = names
		return
	}
	s.Postgres = namesFromText(res.Stdout)
}

func applyValkeyList(s *Snapshot, res Result) {
	if names := nameFromJSON(res.Stdout, "cluster_name", "name"); names != nil {
		s.Valkey = names
		return
	}
	s.Valkey = namesFromText(res.Stdout)
}

func applyNATSList(s *Snapshot, res Result) {
	if names := nameFromJSON(res.Stdout, "app_label", "name"); names != nil {
		s.NATS = names
		return
	}
	s.NATS = namesFromText(res.Stdout)
}

func applyDomainsList(s *Snapshot, res Result) {
	if names := nameFromJSON(res.Stdout, "domain_name", "name"); names != nil {
		s.Domains = names
		return
	}
	s.Domains = namesFromText(res.Stdout)
}

func applyCreditsBalance(s *Snapshot, res Result) {
	var balance struct {
		Balance  float64 `json:"balance"`
		Currency string  `json:"currency"`
	}
	if json.Unmarshal([]byte(res.Stdout), &balance) == nil && balance.Currency != "" {
		s.Credits = fmt.Sprintf("$%.2f %s", balance.Balance, balance.Currency)
		return
	}
	if key, val, ok := splitStatusLine(res.Stdout); ok && key == "Balance" {
		s.Credits = val
	}
}

// namesFromText parses names loosely from table-style text output: the
// first token of each line, skipping headers, dividers and "No X found"
// messages.
func namesFromText(stdout string) []string {
	var names []string
	seen := map[string]bool{}
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "─") || strings.HasPrefix(line, "•") {
			continue
		}
		if strings.Contains(line, "No ") && strings.Contains(line, "found") {
			continue
		}
		first := strings.Fields(line)[0]
		if first == "" || headerWords[first] || seen[first] {
			continue
		}
		seen[first] = true
		names = append(names, first)
	}
	return names
}

// headerWords are first-column words that never name a resource.
var headerWords = map[string]bool{
	"APP": true, "NAME": true, "STORAGE": true, "ID": true, "PROFILE": true,
	"Profiles": true, "API": true, "USER": true, "STATUS": true, "TYPE": true,
	"VERSION": true, "ZONE": true, "DOMAIN": true, "CREATED": true, "REPLICAS": true,
	"Balance": true, "Organization": true, "Authenticated": true, "User": true,
	"Token": true, "Namespace": true,
}
