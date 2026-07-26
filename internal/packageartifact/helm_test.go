package packageartifact

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateHelmEmbedsChartDeterministically(t *testing.T) {
	chart := writeHelmChart(t, map[string]string{
		"Chart.yaml": `apiVersion: v2
name: demo
version: 1.2.3
description: Deterministic demo
annotations:
  satusky.com/supported-architectures: arm64,amd64
  satusky.com/required-secrets: databaseRootPassword,databasePassword,databasePassword
`,
		"values.schema.json": `{"type":"object","properties":{"message":{"type":"string"}},"additionalProperties":false}`,
		"templates/deployment.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Release.Name }}
spec:
  selector:
    matchLabels: {app: demo}
  template:
    metadata:
      labels: {app: demo}
    spec:
      containers:
        - name: demo
          image: registry.example.test/demo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
`,
	})
	first, name, err := CreateHelm(chart)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := CreateHelm(chart)
	if err != nil {
		t.Fatal(err)
	}
	if name != "demo" || !bytes.Equal(first, second) {
		t.Fatalf("CreateHelm() name=%q deterministic=%t", name, bytes.Equal(first, second))
	}
	if archiveName, err := ArchivePackageName(first); err != nil || archiveName != "demo" {
		t.Fatalf("ArchivePackageName() = %q, %v; want demo", archiveName, err)
	}
	files := readArchive(t, first)
	for _, name := range []string{"package.yaml", "values.schema.json", "chart/Chart.yaml", "chart/templates/deployment.yaml"} {
		if files[name] == "" {
			t.Errorf("archive is missing %s", name)
		}
	}
	manifest := files["package.yaml"]
	for _, want := range []string{"version: \"1.2.3\"", "description: \"Deterministic demo\"", "deploymentDriver: helm-renderer", "template: chart", "- amd64", "- arm64", "- \"apps/v1\"", "requiredSecrets:\n      - \"databasePassword\"\n      - \"databaseRootPassword\""} {
		if !strings.Contains(manifest, want) {
			t.Errorf("package manifest does not contain %q:\n%s", want, manifest)
		}
	}
	var schema struct {
		Properties map[string]struct {
			Type    string `json:"type"`
			Minimum int    `json:"minimum"`
			Default int    `json:"default"`
		} `json:"properties"`
	}
	if err := json.Unmarshal([]byte(files["values.schema.json"]), &schema); err != nil {
		t.Fatalf("unmarshal root values schema: %v", err)
	}
	replicas, ok := schema.Properties["replicas"]
	if !ok || replicas.Type != "integer" || replicas.Minimum != 1 || replicas.Default != 1 {
		t.Fatalf("root values schema must normalize replicas to one or more: %#v", replicas)
	}
	if _, ok := schema.Properties["message"]; !ok {
		t.Fatal("root values schema discarded chart input properties")
	}
	withoutSecrets := writeHelmChart(t, map[string]string{
		"Chart.yaml":               "apiVersion: v2\nname: no-secrets\nversion: 1.2.3\nannotations:\n  satusky.com/supported-architectures: amd64\n",
		"templates/configmap.yaml": "apiVersion: v1\nkind: ConfigMap\n",
	})
	archive, _, err := CreateHelm(withoutSecrets)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(readArchive(t, archive)["package.yaml"], "requiredSecrets:") {
		t.Fatal("manifest unexpectedly declares required secrets")
	}
}

func TestCreateHelmRejectsUnsafeOrUnsupportedCharts(t *testing.T) {
	validChart := `apiVersion: v2
name: demo
version: 1.2.3
annotations:
  satusky.com/supported-architectures: amd64
`
	tests := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{name: "missing architectures", files: map[string]string{"Chart.yaml": "apiVersion: v2\nname: demo\nversion: 1.2.3\n", "templates/configmap.yaml": "apiVersion: v1\nkind: ConfigMap\n"}, want: helmArchitecturesAnnotation},
		{name: "invalid required secret", files: map[string]string{"Chart.yaml": validChart + "  satusky.com/required-secrets: 1invalid\n", "templates/configmap.yaml": "apiVersion: v1\nkind: ConfigMap\n"}, want: "invalid secret name"},
		{name: "dependency", files: map[string]string{"Chart.yaml": validChart + "dependencies:\n  - name: child\n    version: 1.0.0\n", "templates/configmap.yaml": "apiVersion: v1\nkind: ConfigMap\n"}, want: "dependencies"},
		{name: "mutable image", files: map[string]string{"Chart.yaml": validChart, "templates/deployment.yaml": "apiVersion: apps/v1\nkind: Deployment\nspec:\n  template:\n    spec:\n      containers:\n        - image: nginx:latest\n"}, want: "mutable image"},
		{name: "lookup", files: map[string]string{"Chart.yaml": validChart, "templates/configmap.yaml": "{{ lookup \"v1\" \"ConfigMap\" \"default\" \"settings\" }}\n"}, want: "nondeterministic"},
		{name: "crd", files: map[string]string{"Chart.yaml": validChart, "crds/widget.yaml": "apiVersion: apiextensions.k8s.io/v1\nkind: CustomResourceDefinition\n"}, want: "CRDs"},
		{name: "hook", files: map[string]string{"Chart.yaml": validChart, "templates/hook.yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  annotations:\n    helm.sh/hook: test\n"}, want: "hooks or tests"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := CreateHelm(writeHelmChart(t, test.files))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("CreateHelm() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestCreateHelmRejectsSymlinks(t *testing.T) {
	chart := writeHelmChart(t, map[string]string{
		"Chart.yaml":               "apiVersion: v2\nname: demo\nversion: 1.2.3\nannotations:\n  satusky.com/supported-architectures: amd64\n",
		"templates/configmap.yaml": "apiVersion: v1\nkind: ConfigMap\n",
	})
	if err := os.Symlink(filepath.Join(chart, "templates", "configmap.yaml"), filepath.Join(chart, "templates", "linked.yaml")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := CreateHelm(chart); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("CreateHelm() error = %v, want symbolic link rejection", err)
	}
}

func writeHelmChart(t *testing.T, files map[string]string) string {
	t.Helper()
	chart := filepath.Join(t.TempDir(), "chart")
	for name, contents := range files {
		filePath := filepath.Join(chart, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(filePath), 0750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filePath, []byte(contents), 0640); err != nil {
			t.Fatal(err)
		}
	}
	return chart
}
