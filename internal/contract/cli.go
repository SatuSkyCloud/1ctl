// Package contract exports stable, machine-readable descriptions of 1ctl.
package contract

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/urfave/cli/v3"
)

const SchemaVersion = 1

type CLIManifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	GeneratedBy   string            `json:"generatedBy"`
	CLI           CLIIdentity       `json:"cli"`
	Commands      []CommandContract `json:"commands"`
}

type CLIIdentity struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type CommandContract struct {
	Path        []string           `json:"path"`
	Name        string             `json:"name"`
	Aliases     []string           `json:"aliases"`
	Usage       string             `json:"usage"`
	Description string             `json:"description,omitempty"`
	Category    string             `json:"category,omitempty"`
	Hidden      bool               `json:"hidden"`
	ArgsUsage   string             `json:"argsUsage"`
	AritySource string             `json:"aritySource"`
	Arguments   []ArgumentContract `json:"arguments"`
	MinArgs     int                `json:"minArgs"`
	MaxArgs     *int               `json:"maxArgs"`
	Flags       []FlagContract     `json:"flags"`
}

type ArgumentContract struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Variadic bool   `json:"variadic"`
}

type FlagContract struct {
	Name       string   `json:"name"`
	Aliases    []string `json:"aliases"`
	Type       string   `json:"type"`
	TakesValue bool     `json:"takesValue"`
	Required   bool     `json:"required"`
	Multiple   bool     `json:"multiple"`
	Default    string   `json:"default,omitempty"`
	Usage      string   `json:"usage"`
	EnvVars    []string `json:"envVars"`
}

// CLI builds a deterministic manifest from the same command tree used by the
// executable. It does not execute command setup hooks or actions.
func CLI(root *cli.Command) (CLIManifest, error) {
	manifest := CLIManifest{
		SchemaVersion: SchemaVersion,
		GeneratedBy:   "go run ./cmd/contractgen",
		CLI: CLIIdentity{
			Name: root.Name,
		},
		Commands: make([]CommandContract, 0),
	}
	if err := appendCommand(&manifest.Commands, root, nil, false); err != nil {
		return CLIManifest{}, err
	}
	return manifest, nil
}

func appendCommand(out *[]CommandContract, cmd *cli.Command, parentPath []string, ancestorHidden bool) error {
	path := append([]string(nil), parentPath...)
	if len(parentPath) > 0 || cmd.Name != "1ctl" {
		path = append(path, cmd.Name)
	}
	if path == nil {
		path = []string{}
	}

	arguments, minArgs, maxArgs, err := parseArgsUsage(cmd.ArgsUsage)
	if err != nil {
		return fmt.Errorf("%s ArgsUsage %q: %w", strings.Join(path, " "), cmd.ArgsUsage, err)
	}
	flags := make([]FlagContract, 0, len(cmd.Flags))
	for _, flag := range cmd.Flags {
		contract, visible := describeFlag(flag)
		if visible {
			flags = append(flags, contract)
		}
	}
	if !cmd.HideHelp {
		flags = append(flags, FlagContract{
			Name:       "help",
			Aliases:    []string{"h"},
			Type:       "boolean",
			TakesValue: false,
			Usage:      "Show help",
			EnvVars:    []string{},
		})
	}
	if len(path) == 0 && !cmd.HideVersion {
		flags = append(flags, FlagContract{
			Name:       "version",
			Aliases:    []string{"v"},
			Type:       "boolean",
			TakesValue: false,
			Usage:      "Print the version",
			EnvVars:    []string{},
		})
	}
	effectivelyHidden := ancestorHidden || cmd.Hidden
	*out = append(*out, CommandContract{
		Path:        path,
		Name:        cmd.Name,
		Aliases:     nonNil(cmd.Aliases),
		Usage:       cmd.Usage,
		Description: cmd.Description,
		Category:    cmd.Category,
		Hidden:      effectivelyHidden,
		ArgsUsage:   cmd.ArgsUsage,
		AritySource: "argsUsage",
		Arguments:   arguments,
		MinArgs:     minArgs,
		MaxArgs:     maxArgs,
		Flags:       flags,
	})

	for _, child := range cmd.Commands {
		if err := appendCommand(out, child, path, effectivelyHidden); err != nil {
			return err
		}
	}
	return nil
}

func describeFlag(flag cli.Flag) (FlagContract, bool) {
	if visible, ok := flag.(cli.VisibleFlag); ok && !visible.IsVisible() {
		return FlagContract{}, false
	}
	doc, ok := flag.(cli.DocGenerationFlag)
	if !ok {
		return FlagContract{}, false
	}
	names := flag.Names()
	if len(names) == 0 {
		return FlagContract{}, false
	}

	required := false
	if value, ok := flag.(cli.RequiredFlag); ok {
		required = value.IsRequired()
	}
	multiple := false
	if value, ok := flag.(cli.DocGenerationMultiValueFlag); ok {
		multiple = value.IsMultiValueFlag()
	}
	flagType := doc.TypeName()
	if value, ok := flag.(cli.SchemaTyper); ok && value.SchemaType() != "" {
		flagType = value.SchemaType()
	}

	defaultValue := doc.GetDefaultText()
	if defaultValue == "" {
		defaultValue = doc.GetValue()
	}
	return FlagContract{
		Name:       names[0],
		Aliases:    nonNil(names[1:]),
		Type:       flagType,
		TakesValue: doc.TakesValue(),
		Required:   required,
		Multiple:   multiple,
		Default:    defaultValue,
		Usage:      doc.GetUsage(),
		EnvVars:    nonNil(doc.GetEnvVars()),
	}, true
}

var argumentTokenPattern = regexp.MustCompile(`^(<[^>]+>|\[[^\]]+\])$`)

func parseArgsUsage(usage string) ([]ArgumentContract, int, *int, error) {
	if strings.TrimSpace(usage) == "" {
		zero := 0
		return []ArgumentContract{}, 0, &zero, nil
	}

	tokens := strings.Fields(usage)
	arguments := make([]ArgumentContract, 0, len(tokens))
	minArgs := 0
	maxArgs := 0
	unbounded := false
	for _, token := range tokens {
		if !argumentTokenPattern.MatchString(token) {
			return nil, 0, nil, fmt.Errorf("unsupported argument token %q", token)
		}
		required := strings.HasPrefix(token, "<")
		name := strings.Trim(token, "<>[]")
		variadic := strings.HasSuffix(name, "...")
		name = strings.TrimSuffix(name, "...")
		if name == "" {
			return nil, 0, nil, fmt.Errorf("argument name is empty")
		}
		arguments = append(arguments, ArgumentContract{
			Name:     name,
			Required: required,
			Variadic: variadic,
		})
		if required {
			minArgs++
		}
		if variadic {
			unbounded = true
		} else {
			maxArgs++
		}
	}
	if unbounded {
		return arguments, minArgs, nil, nil
	}
	return arguments, minArgs, &maxArgs, nil
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
