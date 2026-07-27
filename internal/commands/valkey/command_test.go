package valkey

import "testing"

func TestCommandTree(t *testing.T) {
	cmd := Command()
	if cmd.Name != "valkey" {
		t.Fatalf("command name = %q, want valkey", cmd.Name)
	}
	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "vk" {
		t.Fatalf("command aliases = %v, want [vk]", cmd.Aliases)
	}

	want := map[string]bool{
		"create":             false,
		"list":               false,
		"get":                false,
		"status":             false,
		"credentials":        false,
		"update":             false,
		"users":              false,
		"rotate-credentials": false,
		"metrics":            false,
		"logs":               false,
		"redeploy":           false,
		"restart":            false,
		"delete":             false,
	}
	for _, subcommand := range cmd.Commands {
		if _, ok := want[subcommand.Name]; ok {
			want[subcommand.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("missing %q subcommand", name)
		}
	}
}

func TestCreateHelpIncludesMachinePlacement(t *testing.T) {
	command := createCommand()
	for _, flag := range command.Flags {
		for _, name := range flag.Names() {
			if name == "machine-id" {
				return
			}
		}
	}
	t.Fatal("create command help is missing --machine-id")
}

func TestValidateUserMutation(t *testing.T) {
	valid := userMutationInput{
		Username:        "worker.api",
		AccessPreset:    "read_write",
		KeyPatterns:     []string{"jobs:*"},
		ChannelPatterns: []string{"events:*"},
	}
	if err := validateUserMutation(valid, true); err != nil {
		t.Fatalf("validateUserMutation(valid) returned %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*userMutationInput)
	}{
		{name: "reserved default user", mutate: func(in *userMutationInput) { in.Username = "default" }},
		{name: "reserved replication user", mutate: func(in *userMutationInput) { in.Username = "replication" }},
		{name: "invalid preset", mutate: func(in *userMutationInput) { in.AccessPreset = "commands" }},
		{name: "raw key ACL prefix", mutate: func(in *userMutationInput) { in.KeyPatterns = []string{"~jobs:*"} }},
		{name: "raw channel ACL prefix", mutate: func(in *userMutationInput) { in.ChannelPatterns = []string{"&events:*"} }},
		{name: "pattern whitespace", mutate: func(in *userMutationInput) { in.KeyPatterns = []string{"jobs: *"} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := valid
			test.mutate(&input)
			if err := validateUserMutation(input, true); err == nil {
				t.Fatal("validateUserMutation() returned nil, want error")
			}
		})
	}
}

func TestValidateCreate(t *testing.T) {
	valid := createInput{
		Name:             "sessions",
		Topology:         "replicated",
		Instances:        3,
		AppendFsync:      "everysec",
		MaxmemoryPolicy:  "allkeys-lru",
		MaxmemoryPercent: 75,
	}
	if err := validateCreate(valid); err != nil {
		t.Fatalf("validateCreate(valid) returned %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*createInput)
	}{
		{name: "invalid name", mutate: func(in *createInput) { in.Name = "Invalid_Name" }},
		{name: "invalid topology", mutate: func(in *createInput) { in.Topology = "cluster" }},
		{name: "standalone replicas", mutate: func(in *createInput) {
			in.Topology = "standalone"
			in.Instances = 2
		}},
		{name: "replicated without replica", mutate: func(in *createInput) { in.Instances = 1 }},
		{name: "invalid fsync", mutate: func(in *createInput) { in.AppendFsync = "sometimes" }},
		{name: "invalid eviction policy", mutate: func(in *createInput) { in.MaxmemoryPolicy = "random" }},
		{name: "unsafe memory percentage", mutate: func(in *createInput) { in.MaxmemoryPercent = 95 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := valid
			test.mutate(&input)
			if err := validateCreate(input); err == nil {
				t.Fatal("validateCreate() returned nil, want error")
			}
		})
	}
}
