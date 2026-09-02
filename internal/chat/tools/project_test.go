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

func TestAnalyzeProjectNodeExpress(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"package.json": `{
  "name": "api-service",
  "version": "1.2.0",
  "packageManager": "pnpm@9.15.0",
  "engines": {"node": ">=20"},
  "scripts": {"dev": "tsx watch src/index.ts", "build": "tsc", "start": "node dist/index.js", "test": "vitest"},
  "dependencies": {"express": "^4.19.0", "pg": "^8.11.0", "ioredis": "^5.4.0", "zod": "^3.23.0"},
  "devDependencies": {"typescript": "^5.4.0"}
}`,
		"Dockerfile": "FROM node:20-alpine\nWORKDIR /app\nCOPY . .\nEXPOSE 3000\nCMD [\"npm\", \"start\"]\n",
		"Taskfile.yml": "version: '3'\ntasks:\n  build:\n    cmds: [pnpm build]\n  test:\n    cmds: [pnpm test]\n",
		".github/workflows/deploy.yml": "name: Deploy\non: [push]\n",
	})
	got := analyze(t, dir)
	wantContains(t, got,
		"stack: node",
		"name=api-service",
		"packageManager=pnpm@9.15.0",
		"engines[node=>=20]",
		`scripts: dev="tsx watch src/index.ts" start="node dist/index.js" build="tsc"`,
		"dependencies: redis; sql db client; web framework",
		"port hints: dockerfile EXPOSE 3000, 3000 (express)",
		"existing satusky.toml: none",
		"existing github workflows: Deploy (deploy.yml)",
		"hint: detected a SQL database client — offer a managed postgres",
		"hint: detected a redis client — offer a managed valkey",
		"taskfile[build,test]",
	)
}

func TestAnalyzeProjectNodeNextDockerExpose(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"package.json": `{"name":"dashboard","scripts":{"dev":"next dev","build":"next build","start":"next start"},"dependencies":{"next":"^14.2.0","react":"^18.3.0","@prisma/client":"^5.0.0"}}`,
		"Dockerfile":   "FROM node:20\nEXPOSE 8080\n",
	})
	got := analyze(t, dir)
	wantContains(t, got, "stack: node", "port hints: dockerfile EXPOSE 8080, 3000 (next)", "sql db client")
	// Dockerfile EXPOSE is authoritative and listed before framework defaults.
	if strings.Index(got, "dockerfile EXPOSE 8080") > strings.Index(got, "3000 (next)") {
		t.Errorf("EXPOSE port should be listed before framework default:\n%s", got)
	}
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
		"stack: go",
		"module=github.com/acme/myapi",
		"go=1.24.0",
		"dependencies: postgres/mysql/sql db client; redis; web framework",
	)
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
	wantContains(t, got, "stack: rust", "crate=turboservice", "dependencies: redis; sql db client; web framework")
}

func TestAnalyzeProjectPython(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"pyproject.toml": `[project]
name = "mlapi"
requires-python = ">=3.11"
dependencies = [
  "fastapi>=0.110",
  "uvicorn[standard]>=0.29",
  "psycopg[binary]>=3.1",
  "redis>=5.0",
]
`,
	})
	got := analyze(t, dir)
	wantContains(t, got, "stack: python", "project=mlapi", "python=>=3.11", "dependencies: redis; sql db client; web framework")
}

func TestAnalyzeProjectPythonRequirementsFallback(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"requirements.txt": "# deps\nflask==3.0.0\nSQLAlchemy==2.0.0\ncelery==5.3.0\n",
	})
	got := analyze(t, dir)
	wantContains(t, got, "stack: python", "dependencies: redis; sql db client; web framework")
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
	wantContains(t, got, "stack: undetected", "existing satusky.toml: name=legacy-app port=8080 memory=256Mi health_path=/health strategy=rolling")
}

func TestAnalyzeProjectEmpty(t *testing.T) {
	dir := writeTree(t, map[string]string{})
	got := analyze(t, dir)
	wantContains(t, got, "stack: undetected", "manifests:", "existing satusky.toml: none")
	if strings.Contains(got, "dependencies:") {
		t.Errorf("empty dir should not list dependencies:\n%s", got)
	}
}

func TestAnalyzeProjectSubdir(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"services/api/package.json": `{"name":"svc","scripts":{"start":"node index.js"},"dependencies":{"fastify":"^4.0.0"}}`,
	})
	ex := NewExecutor(dir, nil)
	got := ex.analyzeProject("services/api")
	wantContains(t, got, "stack: node", "name=svc", "port hints: 3000 (fastify)")
}

func TestAnalyzeProjectTraversalRejected(t *testing.T) {
	dir := writeTree(t, map[string]string{"package.json": `{"name":"inner"}`})
	ex := NewExecutor(dir, nil)
	if got := ex.analyzeProject("../.."); !strings.Contains(got, "error:") {
		t.Errorf("traversal path should error, got:\n%s", got)
	}
}

func TestAnalyzeProjectLargeManifestCapped(t *testing.T) {
	big := strings.Repeat("dep", 30_000) // > 64KB
	dir := writeTree(t, map[string]string{"requirements.txt": big})
	got := analyze(t, dir)
	if len(got) > maxProjectReportBytes+200 {
		t.Errorf("report too large: %d bytes", len(got))
	}
	if !strings.Contains(got, "stack: python") {
		t.Errorf("requirements fixture should still detect python:\n%s", got[:200])
	}
}

func TestExecutorDispatchAnalyzeProject(t *testing.T) {
	dir := writeTree(t, map[string]string{"package.json": `{"name":"x","dependencies":{"express":"^4"}}`})
	ex := NewExecutor(dir, nil)
	got := ex.Execute("analyze_project", nil)
	wantContains(t, got, "stack: node", "name=x")
	if strings.Contains(got, "error:") {
		t.Errorf("dispatch should not error:\n%s", got)
	}
	// Malformed args are tolerated (report ignores them).
	bad := ex.Execute("analyze_project", []byte(`{"path": 42}`))
	if !strings.Contains(bad, "error: invalid arguments") {
		t.Errorf("malformed args should error, got:\n%s", bad)
	}
}
