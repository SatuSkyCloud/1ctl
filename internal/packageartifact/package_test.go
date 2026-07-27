package packageartifact

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"
	"text/template"

	"1ctl/internal/config"
)

const testImage = "registry.example.test/acme/demo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestCreateIsDeterministicAndDoesNotEmbedSecretValues(t *testing.T) {
	delay := int32(5)
	project := &config.ProjectConfig{
		App:     config.AppConfig{Name: "demo", Port: 8080, CPURequest: "250m", CPULimit: "1", Memory: "256Mi", Replicas: 2},
		Build:   config.BuildConfig{Image: testImage, TargetArch: "amd64"},
		Secrets: config.SecretsConfig{Required: []string{"API_TOKEN"}},
		Checks:  config.ChecksConfig{Startup: &config.ProbeConfig{HTTPGet: &config.HTTPGetProbeConfig{Path: "/startup", Port: 8080}, PeriodSeconds: &delay}},
		Volumes: []config.VolumeConfig{{Name: "data", Claim: "demo-data", Size: "1Gi", Mount: "/data", StorageClass: "ceph-block"}},
	}
	first, name, err := Create(project, "")
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := Create(project, "")
	if err != nil {
		t.Fatal(err)
	}
	if name != "demo" {
		t.Fatalf("package name = %q, want demo", name)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("package archive bytes are not deterministic")
	}
	files := readArchive(t, first)
	if len(files) != 3 || files["package.yaml"] == "" || files["deploy.yaml"] == "" || files["values.schema.json"] == "" {
		t.Fatalf("package files = %#v, want exactly package.yaml, deploy.yaml, values.schema.json", files)
	}
	if archiveName, err := ArchivePackageName(first); err != nil || archiveName != "demo" {
		t.Fatalf("ArchivePackageName() = %q, %v; want demo", archiveName, err)
	}
	if strings.Contains(strings.Join([]string{files["package.yaml"], files["deploy.yaml"], files["values.schema.json"]}, "\n"), "super-secret-value") {
		t.Fatal("archive contains a secret value")
	}
	if !strings.Contains(files["package.yaml"], "requiredSecrets:\n      - \"API_TOKEN\"") {
		t.Fatalf("manifest did not preserve required secret name:\n%s", files["package.yaml"])
	}
	for _, want := range []string{"image: \"" + testImage + "\"", "startupProbe:", "claimName: \"demo-data\"", "kind: Service", "kind: Ingress", "kind: HTTPRoute"} {
		if !strings.Contains(files["deploy.yaml"], want) {
			t.Errorf("deploy.yaml does not contain %q", want)
		}
	}
}

func TestCreateRendersBothDefaultDNSRoutingVariants(t *testing.T) {
	project := &config.ProjectConfig{
		App:   config.AppConfig{Name: "demo", Port: 8080},
		Build: config.BuildConfig{Image: testImage, TargetArch: "arm64"},
	}
	archive, _, err := Create(project, "")
	if err != nil {
		t.Fatal(err)
	}
	files := readArchive(t, archive)
	if !strings.Contains(files["package.yaml"], "gateway.networking.k8s.io/v1") {
		t.Fatal("package manifest does not declare the Gateway API used by the routing branch")
	}

	gateway := renderDeployTemplate(t, files["deploy.yaml"], map[string]any{
		"AppName": "demo-instance", "Namespace": "tenant-a", "DomainName": "demo.example.com",
		"GatewayMode": true, "GatewayName": "satusky-gateway", "GatewayNamespace": "gateway-system",
		"Metadata": map[string]any{"replicas": 1, "cpu": "250m", "cpuLimit": "1", "memory": "256Mi", "memoryLimit": "256Mi", "port": 8080},
	})
	for _, want := range []string{"kind: HTTPRoute", "name: satusky-gateway", "namespace: gateway-system", "hostnames:", "- demo.example.com", "port: 8080"} {
		if !strings.Contains(gateway, want) {
			t.Errorf("Gateway API render does not contain %q:\n%s", want, gateway)
		}
	}
	if strings.Contains(gateway, "kind: Ingress") {
		t.Errorf("Gateway API render unexpectedly contains Ingress:\n%s", gateway)
	}

	ingress := renderDeployTemplate(t, files["deploy.yaml"], map[string]any{
		"AppName": "demo-instance", "Namespace": "tenant-a", "DomainName": "demo.example.com",
		"GatewayMode": false,
		"Metadata":    map[string]any{"replicas": 1, "cpu": "250m", "cpuLimit": "1", "memory": "256Mi", "memoryLimit": "256Mi", "port": 8080},
	})
	if !strings.Contains(ingress, "kind: Ingress") || strings.Contains(ingress, "kind: HTTPRoute") {
		t.Errorf("Ingress render did not select only the Ingress branch:\n%s", ingress)
	}
}

func renderDeployTemplate(t *testing.T, source string, data map[string]any) string {
	t.Helper()
	tmpl, err := template.New("deploy.yaml").Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		t.Fatal(err)
	}
	return output.String()
}

func TestCreateRejectsMutableImagesAndUnrepresentableValues(t *testing.T) {
	project := &config.ProjectConfig{App: config.AppConfig{Name: "demo"}, Build: config.BuildConfig{Image: "registry.example.test/demo:latest"}}
	if _, _, err := Create(project, ""); err == nil || !strings.Contains(err.Error(), "digest-pinned") {
		t.Fatalf("Create() error = %v, want immutable image error", err)
	}
	project.Build.Image = testImage
	project.Env = config.EnvConfig{"TOKEN": "super-secret-value"}
	if _, _, err := Create(project, ""); err == nil || !strings.Contains(err.Error(), "will not embed [env]") {
		t.Fatalf("Create() error = %v, want env rejection", err)
	}
}

func readArchive(t *testing.T, archive []byte) map[string]string {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	files := make(map[string]string)
	for tarReader := tar.NewReader(reader); ; {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		contents, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatal(err)
		}
		files[strings.TrimPrefix(header.Name, "demo/")] = string(contents)
	}
	return files
}
