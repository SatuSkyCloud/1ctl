package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTree creates a fixture file tree under a temp dir and returns the
// dir.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func analyze(t *testing.T, dir string) string {
	t.Helper()
	ex := NewExecutor(dir, nil)
	return ex.analyzeProject("")
}

func wantContains(t *testing.T, got string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("report missing %q\nreport:\n%s", w, got)
		}
	}
}

func wantNotContains(t *testing.T, got string, notWant ...string) {
	t.Helper()
	for _, w := range notWant {
		if strings.Contains(got, w) {
			t.Errorf("report should not contain %q\nreport:\n%s", w, got)
		}
	}
}

// The analyzer reports FACTS only. It must never impose interpretation —
// no stack labels, no framework->port tables, no dependency categories.
func TestAnalyzeProjectReportsFactsNotInterpretation(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"package.json": `{
  "name": "api-service",
  "packageManager": "pnpm@9.15.0",
  "engines": {"node": ">=20"},
  "scripts": {"dev": "tsx watch src/index.ts", "build": "tsc", "start": "node dist/index.js"},
  "dependencies": {"express": "^4.19.0", "pg": "^8.11.0", "ioredis": "^5.4.0"},
  "devDependencies": {"typescript": "^5.4.0"}
}`,
		"Dockerfile": "FROM node:20-alpine\nWORKDIR /app\nCOPY . .\nEXPOSE 3000\nCMD [\"npm\", \"start\"]\n",
	})
	got := analyze(t, dir)
	wantContains(t, got,
		"manifests: Dockerfile, package.json",
		"package.json: name=api-service packageManager=pnpm@9.15.0 engines[node=>=20]",
		`package.json scripts: dev="tsx watch src/index.ts" start="node dist/index.js" build="tsc"`,
		"package.json dependencies: express, ioredis, pg",
		"package.json devDependencies: typescript",
		"Dockerfile: FROM node:20-alpine EXPOSE 3000 CMD [\"npm\", \"start\"]",
	)
	// Facts, not interpretation: no stack label, no "web framework", no
	// "redis" or "sql db client" categorization, no implied port.
	wantNotContains(t, got,
		"stack:", "web framework", "sql db client", "redis client",
		"port hints", "3000 (express)", "managed postgres", "managed valkey",
	)
}

func TestAnalyzeProjectGo(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"go.mod": `module github.com/acme/myapi

go 1.24.0

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/jackc/pgx/v5 v5.6.0
	github.com/redis/go-redis/v9 v9.5.0
)
`,
	})
	got := analyze(t, dir)
	wantContains(t, got,
		"manifests: go.mod",
		"go.mod: module=github.com/acme/myapi go=1.24.0 requires: github.com/gin-gonic/gin, github.com/jackc/pgx/v5, github.com/redis/go-redis/v9",
	)
	wantNotContains(t, got, "stack:", "web framework", "sql db client")
}

func TestAnalyzeProjectRust(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"Cargo.toml": `[package]
name = "turboservice"
edition = "2021"

[dependencies]
tokio = { version = "1", features = ["full"] }
axum = "0.7"
sqlx = { version = "0.8", features = ["postgres"] }
redis = "0.25"
`,
	})
	got := analyze(t, dir)
	wantContains(t, got, "manifests: Cargo.toml", "Cargo.toml: crate=turboservice deps: axum, redis, sqlx, tokio")
	wantNotContains(t, got, "stack:", "web framework")
}

func TestAnalyzeProjectPython(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"pyproject.toml": `[project]
name = "mlapi"
requires-python = ">=3.11"
dependencies = ["fastapi>=0.110", "uvicorn[standard]>=0.29", "psycopg[binary]>=3.1"]
`,
		"requirements.txt": "flask==3.0.0\nSQLAlchemy==2.0.0\n",
	})
	got := analyze(t, dir)
	wantContains(t, got,
		"pyproject.toml: project=mlapi python=>=3.11 deps: fastapi>=0.110, uvicorn[standard]>=0.29, psycopg[binary]>=3.1",
		"requirements.txt: SQLAlchemy, flask",
	)
	wantNotContains(t, got, "stack:", "web framework", "sql db client")
}

func TestAnalyzeProjectBuildTargetsAndPins(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"Taskfile.yml": "version: '3'\ntasks:\n  build:\n    cmds: [go build]\n  test:\n    cmds: [go test]\n",
		".nvmrc":       "20\n",
	})
	got := analyze(t, dir)
	wantContains(t, got, "taskfile targets: build, test", ".nvmrc: 20")
}

func TestAnalyzeProjectExistingSatusky(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"satusky.toml": `[app]
name = "legacy-app"
port = 8080
memory = "256Mi"

[checks]
health_path = "/health"

[deploy]
strategy = "rolling"
`,
	})
	got := analyze(t, dir)
	wantContains(t, got,
		"manifests: satusky.toml",
		"existing satusky.toml: name=legacy-app port=8080 memory=256Mi health_path=/health strategy=rolling",
	)
}

func TestAnalyzeProjectEmpty(t *testing.T) {
	dir := writeTree(t, map[string]string{})
	got := analyze(t, dir)
	wantContains(t, got, "no manifest files detected")
}

func TestAnalyzeProjectSubdir(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"services/api/package.json": `{"name":"svc","scripts":{"start":"node index.js"},"dependencies":{"fastify":"^4.0.0"}}`,
	})
	ex := NewExecutor(dir, nil)
	got := ex.analyzeProject("services/api")
	wantContains(t, got, "package.json: name=svc", "package.json dependencies: fastify")
}

func TestAnalyzeProjectTraversalRejected(t *testing.T) {
	dir := writeTree(t, map[string]string{"package.json": `{"name":"inner"}`})
	ex := NewExecutor(dir, nil)
	if got := ex.analyzeProject("../.."); !strings.Contains(got, "error:") {
		t.Errorf("traversal path should error, got:\n%s", got)
	}
}

func TestAnalyzeProjectLargeManifestCapped(t *testing.T) {
	big := strings.Repeat("dep==1.0\n", 10_000) // > 64KB
	dir := writeTree(t, map[string]string{"requirements.txt": big})
	got := analyze(t, dir)
	if len(got) > maxProjectReportBytes+200 {
		t.Errorf("report too large: %d bytes", len(got))
	}
	wantContains(t, got, "requirements.txt:")
}

func TestExecutorDispatchAnalyzeProject(t *testing.T) {
	dir := writeTree(t, map[string]string{"package.json": `{"name":"x","dependencies":{"express":"^4"}}`})
	ex := NewExecutor(dir, nil)
	got := ex.Execute("analyze_project", nil)
	wantContains(t, got, "package.json: name=x", "package.json dependencies: express")
	if strings.Contains(got, "error:") {
		t.Errorf("dispatch should not error:\n%s", got)
	}
	// Malformed args are tolerated (report ignores them).
	bad := ex.Execute("analyze_project", []byte(`{"path": 42}`))
	if !strings.Contains(bad, "error: invalid arguments") {
		t.Errorf("malformed args should error, got:\n%s", bad)
	}
}
