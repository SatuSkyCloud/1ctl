package nats

import (
	"context"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestCreateFlagDefaultsAndDestinations(t *testing.T) {
	command := createCommand()
	wantStrings := map[string]string{
		flagCPU:         "250m",
		flagMemory:      "256Mi",
		flagStorageSize: "10Gi",
	}
	for _, flag := range command.Flags {
		switch typed := flag.(type) {
		case *cli.StringFlag:
			if typed.Destination == nil {
				t.Errorf("flag %q has nil Destination", typed.Name)
			}
			if want, ok := wantStrings[typed.Name]; ok && typed.Value != want {
				t.Errorf("flag %q default = %q, want %q", typed.Name, typed.Value, want)
			}
		case *cli.BoolFlag:
			if typed.Destination == nil {
				t.Errorf("flag %q has nil Destination", typed.Name)
			}
		}
	}
}

func TestCreateValidationRejectsUnsafeProfiles(t *testing.T) {
	valid := createInput{Name: "orders", CPU: "250m", Memory: "256Mi", StorageSize: "10Gi"}
	tests := []struct {
		name string
		edit func(*createInput)
		want string
	}{
		{"invalid name", func(in *createInput) { in.Name = "Orders!" }, "name must"},
		{"invalid cpu", func(in *createInput) { in.CPU = "1" }, "--cpu"},
		{"invalid memory", func(in *createInput) { in.Memory = "256M" }, "--memory"},
		{"invalid storage", func(in *createInput) { in.StorageSize = "0Gi" }, "--storage-size"},
		{"core storage override", func(in *createInput) { in.StorageSizeSet = true }, "require --jetstream"},
		{"core storage class", func(in *createInput) { in.StorageClass = "ceph-block" }, "require --jetstream"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			in := valid
			test.edit(&in)
			err := validateCreateInput(in)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestCoreProfileValuesAreExact(t *testing.T) {
	config := natsProfileConfig(false, "10Gi", "")
	cluster := config["cluster"].(map[string]interface{})
	jetstream := config["jetstream"].(map[string]interface{})
	pvc := jetstream["fileStore"].(map[string]interface{})["pvc"].(map[string]interface{})
	if cluster["enabled"] != false || cluster["replicas"] != 1 ||
		jetstream["enabled"] != false || pvc["enabled"] != false ||
		pvc["size"] != "10Gi" || pvc["storageClassName"] != "" {
		t.Fatalf("core profile = %#v", config)
	}
}

func TestCreateCommandParsesJetStreamProfile(t *testing.T) {
	err := Command().Run(context.Background(), []string{
		"nats", "create", "Orders!", "--jetstream", "--storage-size", "20Gi",
	})
	if err == nil || !strings.Contains(err.Error(), "name must") {
		t.Fatalf("error = %v, want name validation", err)
	}
}

func TestCredentialsRequiresOneOutputMode(t *testing.T) {
	for _, in := range []credentialsInput{
		{Deployment: "orders"},
		{Deployment: "orders", OutputDir: "creds", Stdout: true},
	} {
		err := handleCredentials(context.Background(), in)
		if err == nil || !strings.Contains(err.Error(), "exactly one") {
			t.Fatalf("error = %v, want output mode validation", err)
		}
	}
}
