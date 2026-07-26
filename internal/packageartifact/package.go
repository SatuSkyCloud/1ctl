// Package packageartifact creates unsigned marketplace package archives from
// the canonical satusky.toml project configuration.
package packageartifact

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"1ctl/internal/config"
)

const Version = "1.0.1"

var packageNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
var immutableImagePattern = regexp.MustCompile(`^.+@sha256:[a-f0-9]{64}$`)

// Create builds a deterministic, unsigned tar.gz marketplace package. Image
// overrides are intentionally explicit: a package is only reproducible when
// it references an immutable image digest.
func Create(project *config.ProjectConfig, imageOverride string) ([]byte, string, error) {
	if project == nil {
		return nil, "", fmt.Errorf("project configuration is required")
	}
	if err := rejectUnsupported(project); err != nil {
		return nil, "", err
	}
	name := strings.TrimSpace(project.App.Name)
	if !packageNamePattern.MatchString(name) {
		return nil, "", fmt.Errorf("[app].name %q must be a lowercase DNS-compatible package name", name)
	}
	image := strings.TrimSpace(imageOverride)
	if image == "" {
		image = strings.TrimSpace(project.Build.Image)
	}
	if !immutableImagePattern.MatchString(image) {
		return nil, "", fmt.Errorf("package image must be immutable and digest-pinned (expected image@sha256:<64 hex characters>)")
	}

	files, err := packageFiles(project, image)
	if err != nil {
		return nil, "", err
	}
	archive, err := deterministicArchive(name, files)
	if err != nil {
		return nil, "", err
	}
	return archive, name, nil
}

// ArchivePackageName reads the single package-directory name from a generated
// artifact. Publication uses this instead of the output filename, which users
// may freely choose with --output.
func ArchivePackageName(archive []byte) (string, error) {
	reader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return "", fmt.Errorf("open package artifact gzip: %w", err)
	}
	defer func() { _ = reader.Close() }()
	var packageName string
	hasManifest := false
	for tarReader := tar.NewReader(reader); ; {
		header, nextErr := tarReader.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return "", fmt.Errorf("read package artifact: %w", nextErr)
		}
		if header.Typeflag != tar.TypeReg || path.Clean(header.Name) != header.Name {
			return "", fmt.Errorf("package artifact contains an unsafe entry")
		}
		root, fileName, found := strings.Cut(header.Name, "/")
		if !found || root == "" || fileName == "" || strings.Contains(fileName, "/") || !packageNamePattern.MatchString(root) {
			return "", fmt.Errorf("package artifact contains an unsafe path")
		}
		if packageName == "" {
			packageName = root
		} else if packageName != root {
			return "", fmt.Errorf("package artifact must contain one package directory")
		}
		if fileName == "package.yaml" {
			hasManifest = true
		}
	}
	if packageName == "" || !hasManifest {
		return "", fmt.Errorf("package artifact must contain package.yaml")
	}
	return packageName, nil
}

func rejectUnsupported(project *config.ProjectConfig) error {
	if project.Build.Dockerfile != "" || project.Build.FastBuild {
		return fmt.Errorf("package create does not support build settings; supply a digest-pinned --image or [build].image")
	}
	if project.App.CPU != "" || project.App.Domain != "" || project.App.Zone != "" || project.App.Organization != "" {
		return fmt.Errorf("package create cannot represent [app] cpu, domain, zone, or organization settings")
	}
	if project.Checks.HealthPath != "" {
		return fmt.Errorf("package create cannot represent [checks].health_path; use an explicit probe")
	}
	if project.Deploy.Strategy != "" || project.Deploy.RollingMaxSurge != "" || project.Deploy.RollingMaxUnavailable != "" || project.Deploy.MachineTag != "" || len(project.Deploy.WaitFor) != 0 {
		return fmt.Errorf("package create cannot represent [deploy] placement or strategy settings")
	}
	if project.HPA.Enabled || project.VPA.Enabled || project.PDB.Enabled || project.Multicluster.Enabled {
		return fmt.Errorf("package create cannot represent autoscaling, disruption budget, or multicluster settings")
	}
	if len(project.Env) != 0 {
		return fmt.Errorf("package create will not embed [env] values; use package inputs after publication")
	}
	return nil
}

func packageFiles(project *config.ProjectConfig, image string) (map[string][]byte, error) {
	manifest := manifestYAML(project)
	deploy, err := deployYAML(project, image)
	if err != nil {
		return nil, err
	}
	schema, err := valuesSchema(project)
	if err != nil {
		return nil, err
	}
	return map[string][]byte{
		"package.yaml":       []byte(manifest),
		"deploy.yaml":        []byte(deploy),
		"values.schema.json": schema,
	}, nil
}

func manifestYAML(project *config.ProjectConfig) string {
	name := project.App.Name
	archs := []string{"amd64", "arm64"}
	if project.Build.TargetArch != "" {
		archs = []string{project.Build.TargetArch}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "apiVersion: marketplace.satusky.com/v1\nkind: MarketplacePackage\nmetadata:\n  name: %s\n  version: %q\n  displayName: %s\n  signature: %q\nspec:\n  template: deploy.yaml\n  lifecycle:\n    deploymentDriver: manifest-bundle\n    inputSchema: values.schema.json\n    stateful: %t\n    scaling:\n      mode: fixed\n      replicas: %d\n", name, Version, yamlString(name), "", len(project.Volumes) > 0, replicas(project))
	if len(project.Volumes) > 0 {
		b.WriteString("    retention:\n      persistentVolumes: retain\n")
	}
	if len(project.Secrets.Required) > 0 {
		b.WriteString("    requiredSecrets:\n")
		for _, secret := range project.Secrets.Required {
			fmt.Fprintf(&b, "      - %s\n", yamlString(secret))
		}
	}
	b.WriteString("  capabilities:\n    minPlatformVersion: \"1.0.0\"\n    requiredAPIs:\n      - v1\n      - apps/v1\n      - networking.k8s.io/v1\n      - gateway.networking.k8s.io/v1\n    supportedArchitectures:\n")
	for _, arch := range archs {
		fmt.Fprintf(&b, "      - %s\n", arch)
	}
	return b.String()
}

func deployYAML(project *config.ProjectConfig, image string) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.AppName}}
  namespace: {{.Namespace}}
  labels:
    app: {{.AppName}}
spec:
  replicas: {{index .Metadata "replicas"}}
  selector:
    matchLabels:
      app: {{.AppName}}
  template:
    metadata:
      labels:
        app: {{.AppName}}
    spec:
      securityContext:
        runAsNonRoot: true
        seccompProfile:
          type: RuntimeDefault
      containers:
        - name: {{.AppName}}
          image: %s
          securityContext:
            runAsNonRoot: true
            allowPrivilegeEscalation: false
            capabilities:
              drop:
                - ALL
          resources:
            requests:
              cpu: {{index .Metadata "cpu"}}
              memory: {{index .Metadata "memory"}}
            limits:
              cpu: {{index .Metadata "cpuLimit"}}
              memory: {{index .Metadata "memoryLimit"}}
          ports:
            - name: http
              containerPort: {{index .Metadata "port"}}
              protocol: TCP
`, yamlString(image))
	if err := appendProbes(&b, project); err != nil {
		return "", err
	}
	if len(project.Volumes) > 0 {
		b.WriteString("          volumeMounts:\n")
		for _, volume := range project.Volumes {
			fmt.Fprintf(&b, "            - name: %s\n              mountPath: %s\n", yamlString(volume.Name), yamlString(volume.Mount))
		}
		b.WriteString("      volumes:\n")
		for _, volume := range project.Volumes {
			fmt.Fprintf(&b, "        - name: %s\n          persistentVolumeClaim:\n            claimName: %s\n", yamlString(volume.Name), yamlString(volume.Claim))
		}
	}
	b.WriteString(`---
apiVersion: v1
kind: Service
metadata:
  name: {{.AppName}}
  namespace: {{.Namespace}}
spec:
  selector:
    app: {{.AppName}}
  ports:
    - name: http
      protocol: TCP
      port: {{index .Metadata "port"}}
      targetPort: {{index .Metadata "port"}}
---
{{if .GatewayMode}}
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: {{.AppName}}-route
  namespace: {{.Namespace}}
  annotations:
    external-dns.alpha.kubernetes.io/cloudflare-proxied: "true"
spec:
  parentRefs:
    - name: {{.GatewayName}}
      namespace: {{.GatewayNamespace}}
  hostnames:
    - {{.DomainName}}
  rules:
    - matches:
        - path:
            type: PathPrefix
            value: /
      backendRefs:
        - name: {{.AppName}}
          port: {{index .Metadata "port"}}
{{else}}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{.AppName}}-ingress
  namespace: {{.Namespace}}
spec:
  rules:
    - host: {{.DomainName}}
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: {{.AppName}}
                port:
                  number: {{index .Metadata "port"}}
`)
	b.WriteString("{{end}}\n")
	for _, volume := range project.Volumes {
		fmt.Fprintf(&b, `---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: %s
  namespace: {{.Namespace}}
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: %s
  resources:
    requests:
      storage: %s
`, yamlString(volume.Claim), yamlString(volume.StorageClass), yamlString(volume.Size))
	}
	return b.String(), nil
}

func appendProbes(b *strings.Builder, project *config.ProjectConfig) error {
	for _, item := range []struct {
		name  string
		probe *config.ProbeConfig
	}{{"startupProbe", project.Checks.Startup}, {"readinessProbe", project.Checks.Readiness}, {"livenessProbe", project.Checks.Liveness}} {
		if item.probe == nil {
			continue
		}
		fmt.Fprintf(b, "          %s:\n", item.name)
		if probe := item.probe; probe.HTTPGet != nil {
			fmt.Fprintf(b, "            httpGet:\n              path: %s\n              port: %d\n", yamlString(probe.HTTPGet.Path), probe.HTTPGet.Port)
		} else if probe.TCPSocket != nil {
			fmt.Fprintf(b, "            tcpSocket:\n              port: %d\n", probe.TCPSocket.Port)
		} else if probe.Exec != nil {
			b.WriteString("            exec:\n              command:\n")
			for _, command := range probe.Exec.Command {
				fmt.Fprintf(b, "                - %s\n", yamlString(command))
			}
		} else {
			return fmt.Errorf("%s has no supported handler", item.name)
		}
		for _, setting := range []struct {
			name  string
			value *int32
		}{{"initialDelaySeconds", item.probe.InitialDelaySeconds}, {"timeoutSeconds", item.probe.TimeoutSeconds}, {"periodSeconds", item.probe.PeriodSeconds}, {"successThreshold", item.probe.SuccessThreshold}, {"failureThreshold", item.probe.FailureThreshold}} {
			if setting.value != nil {
				fmt.Fprintf(b, "            %s: %d\n", setting.name, *setting.value)
			}
		}
	}
	return nil
}

func valuesSchema(project *config.ProjectConfig) ([]byte, error) {
	values := map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"replicas":    map[string]any{"type": "integer", "minimum": 1, "default": replicas(project)},
			"cpu":         map[string]any{"type": "string", "default": resourceValue(project.App.CPURequest, "250m")},
			"cpuLimit":    map[string]any{"type": "string", "default": resourceValue(project.App.CPULimit, "1")},
			"memory":      map[string]any{"type": "string", "default": resourceValue(project.App.Memory, "256Mi")},
			"memoryLimit": map[string]any{"type": "string", "default": resourceValue(project.App.Memory, "256Mi")},
			"port":        map[string]any{"type": "integer", "minimum": 1, "maximum": 65535, "default": port(project)},
		},
		"required": []string{"replicas", "cpu", "cpuLimit", "memory", "memoryLimit", "port"},
	}
	return json.MarshalIndent(values, "", "  ")
}

func deterministicArchive(packageName string, files map[string][]byte) ([]byte, error) {
	names := make([]string, 0, len(files))
	for name := range files {
		if path.Base(name) != name || name == "" {
			return nil, fmt.Errorf("unsafe package file name %q", name)
		}
		names = append(names, name)
	}
	sort.Strings(names)
	var output bytes.Buffer
	gzipWriter := gzip.NewWriter(&output)
	gzipWriter.Header.ModTime = time.Unix(0, 0).UTC()
	gzipWriter.Header.OS = 255
	tarWriter := tar.NewWriter(gzipWriter)
	for _, name := range names {
		content := files[name]
		if err := tarWriter.WriteHeader(&tar.Header{Name: packageName + "/" + name, Mode: 0640, Typeflag: tar.TypeReg, Size: int64(len(content)), ModTime: time.Unix(0, 0).UTC(), Format: tar.FormatUSTAR}); err != nil {
			return nil, fmt.Errorf("write package file %q: %w", name, err)
		}
		if _, err := tarWriter.Write(content); err != nil {
			return nil, fmt.Errorf("write package file %q: %w", name, err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		return nil, fmt.Errorf("close package tar: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("close package gzip: %w", err)
	}
	return output.Bytes(), nil
}

func yamlString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func resourceValue(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func replicas(project *config.ProjectConfig) int {
	if project.App.Replicas > 0 {
		return project.App.Replicas
	}
	return 1
}

func port(project *config.ProjectConfig) int {
	if project.App.Port > 0 {
		return project.App.Port
	}
	return 8080
}
