// Command contractgen creates the machine-readable artifacts consumed by
// satu-docs and other typed clients.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"1ctl/internal/cliapp"
	"1ctl/internal/contract"

	"github.com/gzuidhof/tygo/tygo"
)

var check = flag.Bool("check", false, "fail if committed contract artifacts are stale")

var apiTypeFiles = []string{
	"admin.go",
	"audit.go",
	"build.go",
	"cluster.go",
	"credits.go",
	"deployment_intent.go",
	"logs.go",
	"marketplace.go",
	"marketplace_publisher.go",
	"models.go",
	"notifications.go",
	"org.go",
	"postgres.go",
	"token.go",
	"user.go",
	"valkey.go",
}

func main() {
	flag.Parse()
	if err := run(*check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(checkOnly bool) error {
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	cliBytes, err := generateCLI()
	if err != nil {
		return err
	}
	apiBytes, err := generateAPITypes(root)
	if err != nil {
		return err
	}

	artifacts := []struct {
		path string
		data []byte
	}{
		{path: filepath.Join(root, "contracts", "cli.json"), data: cliBytes},
		{path: filepath.Join(root, "contracts", "api-types.ts"), data: apiBytes},
	}
	for _, artifact := range artifacts {
		if checkOnly {
			if err := checkArtifact(artifact.path, artifact.data); err != nil {
				return err
			}
			continue
		}
		if err := writeArtifact(artifact.path, artifact.data); err != nil {
			return err
		}
		fmt.Printf("generated %s\n", artifact.path)
	}
	return nil
}

func generateCLI() ([]byte, error) {
	manifest, err := contract.CLI(cliapp.New())
	if err != nil {
		return nil, fmt.Errorf("generate CLI contract: %w", err)
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode CLI contract: %w", err)
	}
	return append(data, '\n'), nil
}

func generateAPITypes(root string) ([]byte, error) {
	if err := validateAPISourceCoverage(root, apiTypeFiles); err != nil {
		return nil, err
	}

	tempDir, err := os.MkdirTemp("", "1ctl-contract-*")
	if err != nil {
		return nil, fmt.Errorf("create API type generation directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir) //nolint:errcheck // Best-effort cleanup of a private temporary directory.
	}()

	outputPath := filepath.Join(tempDir, "api-types.ts")
	config := &tygo.Config{
		TypeMappings: map[string]string{
			"time.Time":     "string /* RFC3339 */",
			"time.Duration": "number /* nanoseconds */",
			"uuid.UUID":     "string /* UUID */",
		},
		Packages: []*tygo.PackageConfig{
			{
				Path:             "1ctl/internal/api",
				OutputPath:       outputPath,
				Indent:           "  ",
				FallbackType:     "unknown",
				Flavor:           "default",
				PreserveComments: "none",
				OptionalType:     "undefined",
				EnumStyle:        "const",
				IncludeFiles:     apiTypeFiles,
			},
		},
	}

	previousDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve current directory: %w", err)
	}
	if err := os.Chdir(root); err != nil {
		return nil, fmt.Errorf("enter repository root: %w", err)
	}
	generateErr := tygo.New(config).Generate()
	restoreErr := os.Chdir(previousDir)
	if generateErr != nil {
		return nil, fmt.Errorf("generate API types: %w", generateErr)
	}
	if restoreErr != nil {
		return nil, fmt.Errorf("restore working directory: %w", restoreErr)
	}

	// outputPath is created inside the private temporary directory above.
	data, err := os.ReadFile(outputPath) // #nosec G304
	if err != nil {
		return nil, fmt.Errorf("read generated API types: %w", err)
	}
	if len(data) == 0 {
		return nil, errors.New("generated API types are empty")
	}
	return data, nil
}

func checkArtifact(path string, expected []byte) error {
	// path is assembled from the discovered repository root and fixed names.
	actual, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		return fmt.Errorf("%s is missing; run go run ./cmd/contractgen: %w", path, err)
	}
	if !bytes.Equal(actual, expected) {
		return fmt.Errorf("%s is stale; run go run ./cmd/contractgen and commit the result", path)
	}
	return nil
}

func writeArtifact(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create contract directory: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".contract-*")
	if err != nil {
		return fmt.Errorf("create temporary contract: %w", err)
	}
	tempPath := temp.Name()
	defer func() {
		_ = os.Remove(tempPath) //nolint:errcheck // Best-effort cleanup after atomic replacement.
	}()

	if _, err := temp.Write(data); err != nil {
		closeErr := temp.Close()
		return errors.Join(fmt.Errorf("write temporary contract: %w", err), closeErr)
	}
	if err := temp.Chmod(0o644); err != nil {
		closeErr := temp.Close()
		return errors.Join(fmt.Errorf("set contract permissions: %w", err), closeErr)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary contract: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace contract artifact: %w", err)
	}
	return nil
}

func repositoryRoot() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	for {
		// current only walks ancestors of the process working directory.
		goMod, readErr := os.ReadFile(filepath.Join(current, "go.mod")) // #nosec G304
		if readErr == nil && strings.HasPrefix(string(goMod), "module 1ctl\n") {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", errors.New("could not find the 1ctl repository root from the working directory")
		}
		current = parent
	}
}

// validateAPISourceCoverage prevents a newly added wire model file from being
// silently omitted by Tygo's explicit file allowlist. Transport-only files may
// stay outside the list as long as they do not declare exported JSON structs.
func validateAPISourceCoverage(root string, includedFiles []string) error {
	apiDirectory := filepath.Join(root, "internal", "api")
	entries, err := os.ReadDir(apiDirectory)
	if err != nil {
		return fmt.Errorf("read API source directory: %w", err)
	}
	included := make(map[string]struct{}, len(includedFiles))
	for _, name := range includedFiles {
		included[name] = struct{}{}
	}

	var omitted []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if _, ok := included[name]; ok {
			continue
		}
		sourcePath := filepath.Join(apiDirectory, name)
		file, parseErr := parser.ParseFile(token.NewFileSet(), sourcePath, nil, 0)
		if parseErr != nil {
			return fmt.Errorf("parse %s: %w", sourcePath, parseErr)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			typeSpec, ok := node.(*ast.TypeSpec)
			if !ok || !typeSpec.Name.IsExported() {
				return true
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return false
			}
			for _, field := range structType.Fields.List {
				if field.Tag == nil {
					continue
				}
				tag, unquoteErr := strconv.Unquote(field.Tag.Value)
				if unquoteErr != nil {
					continue
				}
				jsonName, ok := reflect.StructTag(tag).Lookup("json")
				if ok && jsonName != "-" {
					omitted = append(omitted, name+":"+typeSpec.Name.Name)
					break
				}
			}
			return false
		})
	}
	if len(omitted) > 0 {
		sort.Strings(omitted)
		return fmt.Errorf(
			"exported JSON API types are not included in TypeScript generation: %s; update apiTypeFiles",
			strings.Join(omitted, ", "),
		)
	}
	return nil
}
