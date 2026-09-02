package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// maxManifestBytes caps each manifest read so a huge lockfile or vendored
// file can never flood the model context.
const maxManifestBytes = 64 * 1024

// maxProjectReportBytes caps the whole analyze_project report.
const maxProjectReportBytes = 4 * 1024

// maxListedDeps caps how many dependency names are listed per manifest.
const maxListedDeps = 30

// analyzeProject inventories the chat working directory (or a
// subdirectory) and reports FACTS about the application: which manifests
// exist and their key fields (scripts, dependencies by name, Dockerfile
// FROM/EXPOSE/CMD, build targets, runtime pins, any existing satusky.toml
// and GitHub workflows). It deliberately does NOT interpret the facts —
// no stack labels, no framework→port tables, no dependency
// classification. Interpreting the report is the model's job. Read-only;
// never prompts.
func (e *Executor) analyzeProject(path string) string {
	root := e.Cwd
	if strings.TrimSpace(path) != "" {
		base, err := resolvePath(e.Cwd, path)
		if err != nil {
			return "error: " + err.Error()
		}
		root = base
	}
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "error: no working directory configured"
		}
		root = cwd
	}
	report := newProjectReport(root)
	return report.render()
}

// projectReport accumulates the analysis of one directory.
type projectReport struct {
	root        string
	manifests   []string
	workflows   []string
	sections    []string // one line per manifest with parsed facts
	exposePorts []string
	scripts     string
	satusky     string
}

func newProjectReport(root string) *projectReport {
	return &projectReport{root: root}
}

// section records a facts line for one manifest.
func (r *projectReport) section(line string) { r.sections = append(r.sections, line) }

// readManifest returns the trimmed content of a file if it exists and is
// under maxManifestBytes; ok=false otherwise.
func (r *projectReport) readManifest(name string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(r.root, name)) // #nosec G304 -- name is a fixed manifest filename under the sandboxed root
	if err != nil {
		return "", false
	}
	if len(data) > maxManifestBytes {
		data = data[:maxManifestBytes]
	}
	return string(data), true
}

// manifest runs readManifest and records the file as present.
func (r *projectReport) manifest(name string) (string, bool) {
	content, ok := r.readManifest(name)
	if ok {
		r.manifests = append(r.manifests, name)
	}
	return content, ok
}

// capStrings truncates a sorted list at maxListedDeps with a count note.
func capStrings(names []string) string {
	sort.Strings(names)
	if len(names) == 0 {
		return ""
	}
	if len(names) <= maxListedDeps {
		return strings.Join(names, ", ")
	}
	return strings.Join(names[:maxListedDeps], ", ") + fmt.Sprintf(" …%d more", len(names)-maxListedDeps)
}

// --- per-manifest parsers (facts only) -----------------------------------

func (r *projectReport) parseGoMod(content string) {
	reModule := regexp.MustCompile(`(?m)^module\s+(\S+)`)
	reGo := regexp.MustCompile(`(?m)^go\s+(\S+)`)
	reDep := regexp.MustCompile(`^\t?([^\s/]+(?:\/[^\s/]+)*)\s+v`)
	parts := []string{}
	if m := reModule.FindStringSubmatch(content); len(m) > 1 {
		parts = append(parts, "module="+m[1])
	}
	if m := reGo.FindStringSubmatch(content); len(m) > 1 {
		parts = append(parts, "go="+m[1])
	}
	var deps []string
	for _, line := range strings.Split(content, "\n") {
		if m := reDep.FindStringSubmatch(line); len(m) > 1 {
			deps = append(deps, m[1])
		}
	}
	if c := capStrings(deps); c != "" {
		parts = append(parts, "requires: "+c)
	}
	if len(parts) > 0 {
		r.section("go.mod: " + strings.Join(parts, " "))
	}
}

func (r *projectReport) parseCargo(content string) {
	var c struct {
		Package struct {
			Name    string `toml:"name"`
			Edition string `toml:"edition"`
		} `toml:"package"`
		Dependencies map[string]any `toml:"dependencies"`
	}
	if err := toml.Unmarshal([]byte(content), &c); err != nil {
		return
	}
	parts := []string{}
	if c.Package.Name != "" {
		parts = append(parts, "crate="+c.Package.Name)
	}
	deps := make([]string, 0, len(c.Dependencies))
	for dep := range c.Dependencies {
		deps = append(deps, dep)
	}
	if c := capStrings(deps); c != "" {
		parts = append(parts, "deps: "+c)
	}
	if len(parts) > 0 {
		r.section("Cargo.toml: " + strings.Join(parts, " "))
	}
}

func (r *projectReport) parsePackageJSON(content string) {
	var p struct {
		Name            string            `json:"name"`
		PackageManager  string            `json:"packageManager"`
		Engines         map[string]string `json:"engines"`
		Scripts         map[string]string `json:"scripts"`
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal([]byte(content), &p); err != nil {
		return
	}
	parts := []string{}
	if p.Name != "" {
		parts = append(parts, "name="+p.Name)
	}
	if p.PackageManager != "" {
		parts = append(parts, "packageManager="+p.PackageManager)
	}
	if len(p.Engines) > 0 {
		keys := make([]string, 0, len(p.Engines))
		for k := range p.Engines {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		vals := make([]string, 0, len(keys))
		for _, k := range keys {
			vals = append(vals, k+"="+p.Engines[k])
		}
		parts = append(parts, "engines["+strings.Join(vals, ", ")+"]")
	}
	if len(parts) > 0 {
		r.section("package.json: " + strings.Join(parts, " "))
	}
	if len(p.Scripts) > 0 {
		keys := []string{"dev", "start", "build", "test", "preview", "lint"}
		var list []string
		for _, k := range keys {
			if v, ok := p.Scripts[k]; ok {
				list = append(list, k+"=\""+v+"\"")
			}
		}
		if len(list) > 0 {
			r.scripts = strings.Join(list, " ")
		}
	}
	deps := make([]string, 0, len(p.Dependencies)+len(p.DevDependencies))
	for k := range p.Dependencies {
		deps = append(deps, k)
	}
	dev := make([]string, 0, len(p.DevDependencies))
	for k := range p.DevDependencies {
		dev = append(dev, k)
	}
	if c := capStrings(deps); c != "" {
		r.section("package.json dependencies: " + c)
	}
	if c := capStrings(dev); c != "" {
		r.section("package.json devDependencies: " + c)
	}
}

func (r *projectReport) parsePyProject(content string) {
	var c struct {
		Project struct {
			Name           string   `toml:"name"`
			RequiresPython string   `toml:"requires-python"`
			Dependencies   []string `toml:"dependencies"`
		} `toml:"project"`
		Tool struct {
			Poetry struct {
				Name    string `toml:"name"`
				Version string `toml:"version"`
			} `toml:"poetry"`
		} `toml:"tool"`
	}
	if err := toml.Unmarshal([]byte(content), &c); err != nil {
		return
	}
	parts := []string{}
	name := c.Project.Name
	if name == "" {
		name = c.Tool.Poetry.Name
	}
	if name != "" {
		parts = append(parts, "project="+name)
	}
	if c.Project.RequiresPython != "" {
		parts = append(parts, "python="+c.Project.RequiresPython)
	}
	if len(c.Project.Dependencies) > 0 {
		parts = append(parts, "deps: "+strings.Join(c.Project.Dependencies, ", "))
	}
	if len(parts) > 0 {
		r.section("pyproject.toml: " + strings.Join(parts, " "))
	}
}

func (r *projectReport) parseRequirements(content string) {
	var deps []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// normalize "name==1.2.3" / "name>=1" / "name[extra]"
		dep := line
		if i := strings.IndexAny(dep, "=<>[! \t"); i > 0 {
			dep = dep[:i]
		}
		if dep != "" {
			deps = append(deps, dep)
		}
	}
	if c := capStrings(deps); c != "" {
		r.section("requirements.txt: " + c)
	}
}

func (r *projectReport) parseDockerfile(content string) {
	parts := []string{}
	reFrom := regexp.MustCompile(`(?mi)^\s*FROM\s+([^\s]+)`)
	if m := reFrom.FindStringSubmatch(content); len(m) > 1 {
		parts = append(parts, "FROM "+m[1])
	}
	reExpose := regexp.MustCompile(`(?mi)^\s*EXPOSE\s+([\d\s]+)`)
	for _, m := range reExpose.FindAllStringSubmatch(content, -1) {
		r.exposePorts = append(r.exposePorts, strings.Fields(m[1])...)
	}
	if len(r.exposePorts) > 0 {
		parts = append(parts, "EXPOSE "+strings.Join(r.exposePorts, " "))
	}
	reCmd := regexp.MustCompile(`(?mi)^\s*(CMD|ENTRYPOINT)\s+(.+)$`)
	if m := reCmd.FindStringSubmatch(content); len(m) > 1 {
		parts = append(parts, m[1]+" "+strings.TrimSpace(m[2]))
	}
	if len(parts) > 0 {
		r.section("Dockerfile: " + strings.Join(parts, " "))
	}
}

// parseMakeLike scans Makefile / Taskfile.yml / Justfile for common
// targets (facts: which targets exist — not what they do).
func (r *projectReport) parseMakeLike(content, label string) {
	targets := []string{"build", "test", "run", "dev", "start", "install", "lint", "fmt"}
	found := []string{}
	for _, t := range targets {
		re := regexp.MustCompile(`(?m)^[ \t]*` + regexp.QuoteMeta(t) + `[ \t]*:`)
		if re.MatchString(content) {
			found = append(found, t)
		}
	}
	if len(found) > 0 {
		r.section(label + " targets: " + strings.Join(found, ", "))
	}
}

func (r *projectReport) parseSatuskyToml(content string) {
	var c struct {
		App struct {
			Name     string `toml:"name"`
			Port     int    `toml:"port"`
			Memory   string `toml:"memory"`
			Replicas int    `toml:"replicas"`
		} `toml:"app"`
		Build struct {
			Dockerfile string `toml:"dockerfile"`
			FastBuild  bool   `toml:"fast_build"`
		} `toml:"build"`
		Checks struct {
			HealthPath string `toml:"health_path"`
		} `toml:"checks"`
		Deploy struct {
			Strategy string   `toml:"strategy"`
			WaitFor  []string `toml:"wait_for"`
		} `toml:"deploy"`
	}
	if err := toml.Unmarshal([]byte(content), &c); err != nil {
		return
	}
	parts := []string{}
	if c.App.Name != "" {
		parts = append(parts, "name="+c.App.Name)
	}
	if c.App.Port > 0 {
		parts = append(parts, fmt.Sprintf("port=%d", c.App.Port))
	}
	if c.App.Memory != "" {
		parts = append(parts, "memory="+c.App.Memory)
	}
	if c.App.Replicas > 0 {
		parts = append(parts, fmt.Sprintf("replicas=%d", c.App.Replicas))
	}
	if c.Build.Dockerfile != "" {
		parts = append(parts, "dockerfile="+c.Build.Dockerfile)
	}
	if c.Checks.HealthPath != "" {
		parts = append(parts, "health_path="+c.Checks.HealthPath)
	}
	if c.Deploy.Strategy != "" {
		parts = append(parts, "strategy="+c.Deploy.Strategy)
	}
	if len(c.Deploy.WaitFor) > 0 {
		parts = append(parts, "wait_for="+strings.Join(c.Deploy.WaitFor, ","))
	}
	r.satusky = strings.Join(parts, " ")
}

// --- report assembly -----------------------------------------------------

// render runs the whole scan and produces the compact facts report.
func (r *projectReport) render() string {
	if content, ok := r.manifest("go.mod"); ok {
		r.parseGoMod(content)
	}
	if content, ok := r.manifest("Cargo.toml"); ok {
		r.parseCargo(content)
	}
	if content, ok := r.manifest("package.json"); ok {
		r.parsePackageJSON(content)
	}
	if content, ok := r.manifest("pyproject.toml"); ok {
		r.parsePyProject(content)
	}
	if content, ok := r.manifest("requirements.txt"); ok {
		r.parseRequirements(content)
	}
	if content, ok := r.manifest("Dockerfile"); ok {
		r.parseDockerfile(content)
	}
	for _, pair := range [][2]string{{"Taskfile.yml", "taskfile"}, {"Justfile", "justfile"}, {"Makefile", "makefile"}} {
		if content, ok := r.manifest(pair[0]); ok {
			r.parseMakeLike(content, pair[1])
		}
	}
	for _, f := range []string{".nvmrc", ".node-version", ".python-version", ".tool-versions"} {
		if content, ok := r.manifest(f); ok {
			if first := strings.TrimSpace(strings.SplitN(content, "\n", 2)[0]); first != "" {
				r.section(f + ": " + first)
			}
		}
	}
	if _, ok := r.manifest("docker-compose.yml"); ok {
		r.section("docker-compose.yml present — the app may depend on services defined here")
	}
	if content, ok := r.manifest("satusky.toml"); ok {
		r.parseSatuskyToml(content)
	}
	r.scanWorkflows()
	return r.finalize()
}

// scanWorkflows lists existing GitHub Actions workflows.
func (r *projectReport) scanWorkflows() {
	dir := filepath.Join(r.root, ".github", "workflows")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	reName := regexp.MustCompile(`(?m)^\s*name:\s*(.+)`)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yml") && !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		name := e.Name()
		if data, err := os.ReadFile(filepath.Join(dir, e.Name())); err == nil { // #nosec G304 -- fixed workflow dir under the sandbox
			if m := reName.FindStringSubmatch(string(data)); len(m) > 1 {
				name = strings.TrimSpace(m[1]) + " (" + e.Name() + ")"
			}
		}
		r.workflows = append(r.workflows, name)
	}
	sort.Strings(r.workflows)
}

// finalize composes the report lines and caps total length.
func (r *projectReport) finalize() string {
	var b strings.Builder
	rel := "."
	if r.root != "" && r.root != "." {
		if cwd, err := os.Getwd(); err == nil {
			if relPath, err := filepath.Rel(cwd, r.root); err == nil {
				rel = relPath
			}
		}
	}
	fmt.Fprintf(&b, "## Project analysis: %s\n", rel)
	sort.Strings(r.manifests)
	if len(r.manifests) == 0 {
		b.WriteString("no manifest files detected (no go.mod, Cargo.toml, package.json, pyproject.toml, requirements.txt, Dockerfile, Taskfile.yml, Justfile, Makefile or satusky.toml at this level)\n")
	} else {
		b.WriteString("manifests: " + strings.Join(r.manifests, ", ") + "\n")
	}
	for _, s := range r.sections {
		b.WriteString(s + "\n")
	}
	if r.scripts != "" {
		b.WriteString("package.json scripts: " + r.scripts + "\n")
	}
	if r.satusky != "" {
		b.WriteString("existing satusky.toml: " + r.satusky + "\n")
	}
	if len(r.workflows) > 0 {
		b.WriteString("existing github workflows: " + strings.Join(r.workflows, ", ") + "\n")
	}
	b.WriteString("next: determine the stack, build/start commands, port and stateful dependencies from the facts above (read files or ask the user for anything not determinable), then decide what the user actually needs (deploy config, CI, managed services, ...).\n")
	out := b.String()
	if len(out) > maxProjectReportBytes {
		out = out[:maxProjectReportBytes] + "\n…(report truncated)"
	}
	return out
}
