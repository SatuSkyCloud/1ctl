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

// analyzeProject inspects the chat working directory (or a subdirectory)
// for the files that describe an application — manifests, build files,
// Dockerfile, existing satusky.toml and CI workflows — and returns a
// compact structured report the model can use to understand the stack,
// its dependencies, and what it needs to ask before generating
// satusky.toml or a GitHub Actions workflow. Read-only; never prompts.
func (e *Executor) analyzeProject(path string) string {
	if strings.TrimSpace(path) == "" {
		// No subdirectory: analyze the chat working directory itself.
		if e.Cwd == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return "error: no working directory configured"
			}
			e.Cwd = cwd
		}
		report := newProjectReport(e.Cwd)
		return report.render()
	}
	base, err := resolvePath(e.Cwd, path)
	if err != nil {
		return "error: " + err.Error()
	}
	report := newProjectReport(base)
	return report.render()
}

// projectReport accumulates the analysis of one directory.
type projectReport struct {
	root       string
	manifests  []string
	workflows  []string
	deps       map[string]bool // categorized dependency hints ("postgres", "redis", ...)
	portHints  []string
	stack      string
	stackExtra string
	satusky    string
	scripts    string
	exposePorts []string
}

func newProjectReport(root string) *projectReport {
	return &projectReport{root: root, deps: map[string]bool{}}
}

func (r *projectReport) addDep(category string) { r.deps[category] = true }

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

// stackHint sets the detected stack if not already known (first match wins
// by priority order of the caller).
func (r *projectReport) stackHint(name string) {
	if r.stack == "" {
		r.stack = name
	}
}

// port records a port hint in detection order (first recorded wins later).
func (r *projectReport) port(hint string) {
	r.portHints = append(r.portHints, hint)
}

// --- per-manifest parsers -------------------------------------------------

func (r *projectReport) parseGoMod(content string) {
	r.stackHint("go")
	reModule := regexp.MustCompile(`(?m)^module\s+(\S+)`)
	reGo := regexp.MustCompile(`(?m)^go\s+(\S+)`)
	reDep := regexp.MustCompile(`^\t?([^\s/]+(?:\/[^\s/]+)*)\s+v`)
	if m := reModule.FindStringSubmatch(content); len(m) > 1 {
		r.stackExtra = "module=" + m[1]
	}
	if m := reGo.FindStringSubmatch(content); len(m) > 1 {
		r.stackExtra += " go=" + m[1]
	}
	for _, line := range strings.Split(content, "\n") {
		if m := reDep.FindStringSubmatch(line); len(m) > 1 {
			r.categorizeGoDep(m[1])
		}
	}
}

func (r *projectReport) categorizeGoDep(mod string) {
	mod = strings.TrimSpace(mod)
	switch {
	case strings.Contains(mod, "jackc/pgx"), strings.Contains(mod, "lib/pq"),
		strings.Contains(mod, "go-sql-driver/mysql"), strings.Contains(mod, "mattn/go-sqlite3"),
		strings.Contains(mod, "mongo-driver"), strings.Contains(mod, "gorm.io"),
		strings.Contains(mod, "jmoiron/sqlx"):
		r.addDep("postgres/mysql/sql db client")
	case strings.Contains(mod, "go-redis"), strings.Contains(mod, "gomodule/redigo"):
		r.addDep("redis")
	case strings.Contains(mod, "gin-gonic"), strings.Contains(mod, "gorilla/mux"),
		strings.Contains(mod, "labstack/echo"), strings.Contains(mod, "gofiber/fiber"),
		strings.Contains(mod, "chi"):
		r.addDep("web framework")
	case strings.Contains(mod, "nats-io"):
		r.addDep("nats")
	}
}

func (r *projectReport) parseCargo(content string) {
	r.stackHint("rust")
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
	if c.Package.Name != "" {
		r.stackExtra = "crate=" + c.Package.Name
	}
	for dep := range c.Dependencies {
		switch dep {
		case "axum", "actix-web", "rocket", "warp", "tonic":
			r.addDep("web framework")
		case "sqlx", "diesel", "rusqlite":
			r.addDep("sql db client")
		case "redis":
			r.addDep("redis")
		}
	}
}

func (r *projectReport) parsePackageJSON(content string) {
	r.stackHint("node")
	var p struct {
		Name            string            `json:"name"`
		Version         string            `json:"version"`
		PackageManager  string            `json:"packageManager"`
		Engines         map[string]string `json:"engines"`
		Scripts         map[string]string `json:"scripts"`
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal([]byte(content), &p); err != nil {
		return
	}
	extra := []string{}
	if p.Name != "" {
		extra = append(extra, "name="+p.Name)
	}
	if p.PackageManager != "" {
		extra = append(extra, "packageManager="+p.PackageManager)
	}
	if len(p.Engines) > 0 {
		parts := make([]string, 0, len(p.Engines))
		for k, v := range p.Engines {
			parts = append(parts, k+"="+v)
		}
		sort.Strings(parts)
		extra = append(extra, "engines["+strings.Join(parts, ", ")+"]")
	}
	r.stackExtra = strings.Join(extra, " ")
	if len(p.Scripts) > 0 {
		keys := []string{"dev", "start", "build", "test", "preview", "lint"}
		var parts []string
		for _, k := range keys {
			if v, ok := p.Scripts[k]; ok {
				parts = append(parts, k+"=\""+v+"\"")
			}
		}
		if len(parts) > 0 {
			r.scripts = strings.Join(parts, " ")
		}
	}
	all := map[string]string{}
	for k, v := range p.Dependencies {
		all[k] = v
	}
	for k, v := range p.DevDependencies {
		all[k] = v
	}
	r.categorizeNodeDeps(all)
}

func (r *projectReport) categorizeNodeDeps(all map[string]string) {
	for dep := range all {
		switch {
		case nodeDB[dep]:
			r.addDep("sql db client")
		case nodeRedis[dep]:
			r.addDep("redis")
		case nodeNATS[dep]:
			r.addDep("nats")
		case nodeFramework[dep]:
			r.addDep("web framework")
			if p, ok := nodeFrameworkPorts[dep]; ok {
				r.port(fmt.Sprintf("%s (%s)", p, dep))
			}
		}
	}
}

var nodeDB = map[string]bool{
	"pg": true, "pg-promise": true, "pg-pool": true, "mysql2": true, "mysql": true,
	"sqlite3": true, "better-sqlite3": true, "mongodb": true, "mongoose": true,
	"prisma": true, "@prisma/client": true, "typeorm": true, "knex": true, "sequelize": true,
}
var nodeRedis = map[string]bool{
	"redis": true, "ioredis": true, "node-redis": true, "bullmq": true, "bull": true,
}
var nodeNATS = map[string]bool{"nats": true}
var nodeFramework = map[string]bool{
	"next": true, "nuxt": true, "sveltekit": true, "@sveltejs/kit": true, "vite": true,
	"react-scripts": true, "express": true, "fastify": true, "@nestjs/core": true,
	"koa": true, "remix": true, "gatsby": true, "astro": true, "@angular/core": true,
	"vue": true, "@vue/cli-service": true,
}
var nodeFrameworkPorts = map[string]string{
	"next": "3000", "nuxt": "3000", "@sveltejs/kit": "5173", "sveltekit": "5173",
	"vite": "5173", "react-scripts": "3000", "express": "3000", "fastify": "3000",
	"@nestjs/core": "3000", "koa": "3000", "remix": "3000", "gatsby": "8000",
	"astro": "4321", "@angular/core": "4200", "vue": "5173", "@vue/cli-service": "8080",
}

func (r *projectReport) parsePyProject(content string) {
	r.stackHint("python")
	var c struct {
		Project struct {
			Name            string   `toml:"name"`
			RequiresPython  string   `toml:"requires-python"`
			Dependencies    []string `toml:"dependencies"`
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
	name := c.Project.Name
	if name == "" {
		name = c.Tool.Poetry.Name
	}
	if name != "" {
		r.stackExtra = "project=" + name
	}
	if c.Project.RequiresPython != "" {
		r.stackExtra += " python=" + c.Project.RequiresPython
	}
	for _, dep := range c.Project.Dependencies {
		r.categorizePythonDep(dep)
	}
}

func (r *projectReport) parseRequirements(content string) {
	r.stackHint("python")
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
			r.categorizePythonDep(dep)
		}
	}
}

func (r *projectReport) categorizePythonDep(dep string) {
	dep = strings.ToLower(strings.TrimSpace(dep))
	switch {
	case strings.HasPrefix(dep, "fastapi"), strings.HasPrefix(dep, "flask"),
		strings.HasPrefix(dep, "django"), strings.HasPrefix(dep, "aiohttp"),
		strings.HasPrefix(dep, "tornado"), strings.HasPrefix(dep, "starlette"),
		strings.HasPrefix(dep, "uvicorn"), strings.HasPrefix(dep, "gunicorn"):
		r.addDep("web framework")
	case strings.HasPrefix(dep, "psycopg"), strings.HasPrefix(dep, "asyncpg"),
		strings.HasPrefix(dep, "pymysql"), strings.HasPrefix(dep, "sqlalchemy"),
		strings.HasPrefix(dep, "pymongo"), strings.HasPrefix(dep, "aiosqlite"):
		r.addDep("sql db client")
	case strings.HasPrefix(dep, "redis"), strings.HasPrefix(dep, "aioredis"),
		strings.HasPrefix(dep, "celery"):
		r.addDep("redis")
	case strings.HasPrefix(dep, "nats-py"), strings.HasPrefix(dep, "nats"):
		r.addDep("nats")
	}
}

func (r *projectReport) parseDockerfile(content string) {
	reExpose := regexp.MustCompile(`(?mi)^\s*EXPOSE\s+(\d+)`)
	for _, m := range reExpose.FindAllStringSubmatch(content, -1) {
		r.exposePorts = append(r.exposePorts, m[1])
	}
	reFrom := regexp.MustCompile(`(?mi)^\s*FROM\s+([^\s]+)`)
	if m := reFrom.FindStringSubmatch(content); len(m) > 1 {
		base := strings.ToLower(m[1])
		switch {
		case strings.Contains(base, "golang"):
			r.stackHint("go (dockerfile)")
		case strings.Contains(base, "node"):
			r.stackHint("node (dockerfile)")
		case strings.Contains(base, "python"):
			r.stackHint("python (dockerfile)")
		case strings.Contains(base, "rust"):
			r.stackHint("rust (dockerfile)")
		}
	}
}

// parseMakeLike scans Makefile / Taskfile.yml / Justfile for common
// targets. It does not try to understand the build system — presence of a
// target is enough for the model to ask the right question. Targets may
// be top-level or indented (Makefile tabs, Taskfile/Justfile nesting).
func (r *projectReport) parseMakeLike(content string, label string) {
	targets := []string{"build", "test", "run", "dev", "start", "install", "lint", "fmt"}
	found := []string{}
	for _, t := range targets {
		re := regexp.MustCompile(`(?m)^[ \t]*` + regexp.QuoteMeta(t) + `[ \t]*:`)
		if re.MatchString(content) {
			found = append(found, t)
		}
	}
	if len(found) > 0 {
		r.stackExtra += " " + label + "[" + strings.Join(found, ",") + "]"
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

// render runs the whole scan and produces the compact text report.
func (r *projectReport) render() string {
	// Priority order for stack detection: go > rust > node > python > others.
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
	// Build orchestrators (presence + common targets only).
	for _, pair := range [][2]string{{"Taskfile.yml", "taskfile"}, {"Justfile", "justfile"}, {"Makefile", "makefile"}} {
		if content, ok := r.manifest(pair[0]); ok {
			r.parseMakeLike(content, pair[1])
		}
	}
	// Runtime version pins.
	for _, f := range []string{".nvmrc", ".node-version", ".python-version", ".tool-versions", "docker-compose.yml"} {
		if content, ok := r.manifest(f); ok {
			first := strings.TrimSpace(strings.SplitN(content, "\n", 2)[0])
			if first != "" {
				r.stackExtra += " " + f + "=" + first
			}
		}
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
	if r.stack != "" {
		fmt.Fprintf(&b, "stack: %s", r.stack)
		if r.stackExtra != "" {
			b.WriteString(" (" + strings.TrimSpace(r.stackExtra) + ")")
		}
		b.WriteString("\n")
	} else {
		b.WriteString("stack: undetected — no go.mod/Cargo.toml/package.json/pyproject.toml/Dockerfile found at this level\n")
	}
	sort.Strings(r.manifests)
	b.WriteString("manifests: " + strings.Join(r.manifests, ", ") + "\n")
	if r.scripts != "" {
		b.WriteString("scripts: " + r.scripts + "\n")
	}
	if len(r.deps) > 0 {
		keys := make([]string, 0, len(r.deps))
		for k := range r.deps {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteString("dependencies: " + strings.Join(keys, "; ") + "\n")
	}
	if len(r.exposePorts) > 0 || len(r.portHints) > 0 {
		// Dockerfile EXPOSE is authoritative (the container must listen
		// there); framework defaults are guesses and come after.
		var ports []string
		for _, p := range r.exposePorts {
			ports = append(ports, "dockerfile EXPOSE "+p)
		}
		ports = append(ports, r.portHints...)
		b.WriteString("port hints: " + strings.Join(ports, ", ") + "\n")
	}
	if r.satusky != "" {
		b.WriteString("existing satusky.toml: " + r.satusky + "\n")
	} else {
		b.WriteString("existing satusky.toml: none\n")
	}
	if len(r.workflows) > 0 {
		b.WriteString("existing github workflows: " + strings.Join(r.workflows, ", ") + "\n")
	}
	if r.deps["sql db client"] {
		b.WriteString("hint: detected a SQL database client — offer a managed postgres instance and a [deploy] wait_for dependency\n")
	}
	if r.deps["redis"] {
		b.WriteString("hint: detected a redis client — offer a managed valkey instance and a [deploy] wait_for dependency\n")
	}
	if r.deps["nats"] {
		b.WriteString("hint: detected a nats client — offer a managed NATS instance\n")
	}
	if r.stack == "" {
		b.WriteString("hint: ask whether a Dockerfile exists or should be generated, and what runtime the app expects\n")
	}
	out := b.String()
	if len(out) > maxProjectReportBytes {
		out = out[:maxProjectReportBytes] + "\n…(report truncated)"
	}
	return out
}
