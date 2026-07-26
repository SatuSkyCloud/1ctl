package packageartifact

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	helmChartDirectory                  = "chart"
	helmArchitecturesAnnotation         = "satusky.com/supported-architectures"
	helmRequiredSecretsAnnotation       = "satusky.com/required-secrets"
	helmMaxChartFiles                   = 96
	helmMaxChartBytes             int64 = 24 << 20
	helmMaxArchiveBytes                 = 8 << 20
)

var helmVersionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-(?:0|[1-9][0-9]*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*))*?)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
var helmUnsupportedTemplateFunction = regexp.MustCompile(`(?s)\{\{[^}]*\b(lookup|now|ago|date(?:InZone|Modify)?|htmlDate(?:InZone)?|rand\w*|uuidv4|gen(?:CA|SelfSignedCert|SignedCert|PrivateKey)|derivePassword|getHostByName|tpl)\b`)
var helmImageReference = regexp.MustCompile(`(?m)^\s*(?:-\s*)?image:\s*["']?([^\s#"']+)`)
var helmAPIVersion = regexp.MustCompile(`(?m)^\s*apiVersion:\s*([^\s#]+)`)
var helmKind = regexp.MustCompile(`(?m)^\s*kind:\s*([^\s#]+)`)
var helmCredentialName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9._-]{0,127}$`)

type helmChartMetadata struct {
	Name         string            `yaml:"name"`
	Version      string            `yaml:"version"`
	Description  string            `yaml:"description"`
	Dependencies []yaml.Node       `yaml:"dependencies"`
	Annotations  map[string]string `yaml:"annotations"`
}

// CreateHelm packages a self-contained Helm chart for the offline
// helm-renderer driver. It deliberately never runs Helm, resolves a chart
// dependency, or reads from the network.
func CreateHelm(chartDir string) ([]byte, string, error) {
	chartDir = filepath.Clean(chartDir)
	metadata, files, apis, stateful, err := readHelmChart(chartDir)
	if err != nil {
		return nil, "", err
	}
	architectures, err := helmArchitectures(metadata.Annotations)
	if err != nil {
		return nil, "", err
	}
	requiredSecrets, err := helmRequiredSecrets(metadata.Annotations)
	if err != nil {
		return nil, "", err
	}
	if !packageNamePattern.MatchString(metadata.Name) {
		return nil, "", fmt.Errorf("Chart.yaml name %q must be a lowercase DNS-compatible package name", metadata.Name)
	}
	if !helmVersionPattern.MatchString(metadata.Version) {
		return nil, "", fmt.Errorf("Chart.yaml version %q must be semantic versioning", metadata.Version)
	}
	if len(apis) == 0 {
		return nil, "", fmt.Errorf("Helm chart templates must declare at least one literal apiVersion")
	}

	schema, err := helmValuesSchema(files)
	if err != nil {
		return nil, "", err
	}
	artifactFiles := map[string][]byte{
		"package.yaml":       []byte(helmManifestYAML(metadata, architectures, requiredSecrets, apis, stateful)),
		"values.schema.json": schema,
	}
	for relative, contents := range files {
		if relative == "values.schema.json" {
			contents = schema
		}
		artifactFiles[helmChartDirectory+"/"+relative] = contents
	}
	archive, err := deterministicArchive(metadata.Name, artifactFiles)
	if err != nil {
		return nil, "", err
	}
	if len(archive) > helmMaxArchiveBytes {
		return nil, "", fmt.Errorf("Helm package archive exceeds %d byte limit", helmMaxArchiveBytes)
	}
	return archive, metadata.Name, nil
}

func readHelmChart(chartDir string) (helmChartMetadata, map[string][]byte, []string, bool, error) {
	info, err := os.Lstat(chartDir)
	if err != nil {
		return helmChartMetadata{}, nil, nil, false, fmt.Errorf("inspect Helm chart: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return helmChartMetadata{}, nil, nil, false, fmt.Errorf("Helm chart path must be a non-symbolic-link directory")
	}
	files := make(map[string][]byte)
	apiSet := make(map[string]struct{})
	stateful := false
	fileCount := 0
	var byteCount int64
	err = filepath.WalkDir(chartDir, func(filePath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if filePath == chartDir {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("Helm chart contains a symbolic link")
		}
		relative, err := filepath.Rel(chartDir, filePath)
		if err != nil || !filepath.IsLocal(relative) {
			return fmt.Errorf("Helm chart contains an unsafe path")
		}
		relative = filepath.ToSlash(relative)
		parts := strings.Split(relative, "/")
		if entry.IsDir() {
			for _, part := range parts {
				if part == "charts" {
					return fmt.Errorf("Helm charts may not contain dependencies")
				}
				if part == "crds" {
					return fmt.Errorf("Helm charts may not contain CRDs")
				}
				if part == "tests" {
					return fmt.Errorf("Helm charts may not contain hooks or tests")
				}
			}
			return nil
		}
		if relative == "Chart.lock" {
			return fmt.Errorf("Helm charts may not contain dependencies")
		}
		if !entry.Type().IsRegular() || !helmArtifactFile(relative) {
			return fmt.Errorf("Helm chart contains an unsupported file %q", relative)
		}
		fileCount++
		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}
		byteCount += fileInfo.Size()
		if fileCount > helmMaxChartFiles || byteCount > helmMaxChartBytes {
			return fmt.Errorf("Helm chart exceeds packaging limits")
		}
		contents, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		if helmUnsupportedTemplateFunction.Match(contents) {
			return fmt.Errorf("Helm chart contains an unsupported or nondeterministic template function")
		}
		if strings.Contains(string(contents), "helm.sh/hook") {
			return fmt.Errorf("Helm charts may not contain hooks or tests")
		}
		if err := rejectMutableHelmImages(contents); err != nil {
			return err
		}
		if strings.HasPrefix(relative, "templates/") {
			for _, match := range helmAPIVersion.FindAllSubmatch(contents, -1) {
				apiVersion := strings.TrimSpace(string(match[1]))
				if strings.Contains(apiVersion, "{{") || apiVersion == "" {
					return fmt.Errorf("Helm chart apiVersion must be literal")
				}
				apiSet[apiVersion] = struct{}{}
			}
			for _, match := range helmKind.FindAllSubmatch(contents, -1) {
				if strings.TrimSpace(string(match[1])) == "PersistentVolumeClaim" {
					stateful = true
				}
			}
		}
		files[relative] = contents
		return nil
	})
	if err != nil {
		return helmChartMetadata{}, nil, nil, false, fmt.Errorf("validate Helm chart: %w", err)
	}
	chartYAML, ok := files["Chart.yaml"]
	if !ok {
		return helmChartMetadata{}, nil, nil, false, fmt.Errorf("Helm chart must contain Chart.yaml")
	}
	var metadata helmChartMetadata
	if err := yaml.Unmarshal(chartYAML, &metadata); err != nil {
		return helmChartMetadata{}, nil, nil, false, fmt.Errorf("parse Chart.yaml: %w", err)
	}
	if len(metadata.Dependencies) > 0 {
		return helmChartMetadata{}, nil, nil, false, fmt.Errorf("Helm charts may not declare dependencies")
	}
	apis := make([]string, 0, len(apiSet))
	for api := range apiSet {
		apis = append(apis, api)
	}
	sort.Strings(apis)
	return metadata, files, apis, stateful, nil
}

func helmArtifactFile(relative string) bool {
	base := path.Base(relative)
	if base == "Chart.yaml" || base == "Chart.yml" || base == "README.md" {
		return true
	}
	extension := strings.ToLower(path.Ext(base))
	return extension == ".yaml" || extension == ".yml" || extension == ".json" || extension == ".tpl" || extension == ".txt" || extension == ".md"
}

func helmArchitectures(annotations map[string]string) ([]string, error) {
	value := strings.TrimSpace(annotations[helmArchitecturesAnnotation])
	if value == "" {
		return nil, fmt.Errorf("Chart.yaml annotations.%s must declare supported architectures as comma-separated amd64 and/or arm64", helmArchitecturesAnnotation)
	}
	seen := make(map[string]struct{})
	architectures := make([]string, 0, 2)
	for _, architecture := range strings.Split(value, ",") {
		architecture = strings.TrimSpace(architecture)
		if architecture != "amd64" && architecture != "arm64" {
			return nil, fmt.Errorf("Chart.yaml annotations.%s has unsupported architecture %q", helmArchitecturesAnnotation, architecture)
		}
		if _, duplicate := seen[architecture]; duplicate {
			return nil, fmt.Errorf("Chart.yaml annotations.%s declares %q more than once", helmArchitecturesAnnotation, architecture)
		}
		seen[architecture] = struct{}{}
		architectures = append(architectures, architecture)
	}
	sort.Strings(architectures)
	return architectures, nil
}

func rejectMutableHelmImages(contents []byte) error {
	for _, match := range helmImageReference.FindAllSubmatch(contents, -1) {
		reference := strings.TrimSpace(string(match[1]))
		if reference == "" || strings.Contains(reference, "{{") {
			continue
		}
		if !immutableImagePattern.MatchString(reference) {
			return fmt.Errorf("Helm chart contains mutable image reference %q; use image@sha256:<64 hex characters>", reference)
		}
	}
	return nil
}

func helmManifestYAML(metadata helmChartMetadata, architectures, requiredSecrets, apis []string, stateful bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "apiVersion: marketplace.satusky.com/v1\nkind: MarketplacePackage\nmetadata:\n  name: %s\n  version: %s\n  displayName: %s\n", metadata.Name, yamlString(metadata.Version), yamlString(metadata.Name))
	if strings.TrimSpace(metadata.Description) != "" {
		fmt.Fprintf(&b, "  description: %s\n", yamlString(metadata.Description))
	}
	b.WriteString("  signature: \"\"\nspec:\n  template: chart\n  lifecycle:\n    deploymentDriver: helm-renderer\n    inputSchema: values.schema.json\n")
	fmt.Fprintf(&b, "    stateful: %t\n    scaling:\n      mode: unlimited\n", stateful)
	if stateful {
		b.WriteString("    retention:\n      persistentVolumes: retain\n")
	}
	if len(requiredSecrets) > 0 {
		b.WriteString("    requiredSecrets:\n")
		for _, secret := range requiredSecrets {
			fmt.Fprintf(&b, "      - %s\n", yamlString(secret))
		}
	}
	b.WriteString("  capabilities:\n    minPlatformVersion: \"1.0.0\"\n    requiredAPIs:\n")
	for _, api := range apis {
		fmt.Fprintf(&b, "      - %s\n", yamlString(api))
	}
	b.WriteString("    supportedArchitectures:\n")
	for _, architecture := range architectures {
		fmt.Fprintf(&b, "      - %s\n", architecture)
	}
	return b.String()
}

func helmRequiredSecrets(annotations map[string]string) ([]string, error) {
	value := strings.TrimSpace(annotations[helmRequiredSecretsAnnotation])
	if value == "" {
		return nil, nil
	}

	seen := make(map[string]struct{})
	for _, name := range strings.Split(value, ",") {
		name = strings.TrimSpace(name)
		if !helmCredentialName.MatchString(name) {
			return nil, fmt.Errorf("Chart.yaml annotation %q has invalid secret name %q", helmRequiredSecretsAnnotation, name)
		}
		seen[name] = struct{}{}
	}

	secrets := make([]string, 0, len(seen))
	for name := range seen {
		secrets = append(secrets, name)
	}
	sort.Strings(secrets)
	return secrets, nil
}

func helmValuesSchema(files map[string][]byte) ([]byte, error) {
	schema, exists := files["values.schema.json"]
	if !exists {
		schema = []byte("{\n  \"$schema\": \"https://json-schema.org/draft/2020-12/schema\",\n  \"type\": \"object\",\n  \"additionalProperties\": true\n}\n")
	}
	var document map[string]any
	if err := json.Unmarshal(schema, &document); err != nil {
		return nil, fmt.Errorf("chart values.schema.json is invalid JSON: %w", err)
	}
	if document["type"] != "object" {
		return nil, fmt.Errorf("chart values.schema.json must have root type object")
	}
	properties, ok := document["properties"].(map[string]any)
	if !ok {
		if _, exists := document["properties"]; exists {
			return nil, fmt.Errorf("chart values.schema.json properties must be an object")
		}
		properties = make(map[string]any)
		document["properties"] = properties
	}
	properties["replicas"] = map[string]any{
		"type":    "integer",
		"minimum": 1,
		"default": 1,
	}
	return json.Marshal(document)
}
