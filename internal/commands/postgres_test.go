package commands

import "testing"

func TestPostgresCommandStructure(t *testing.T) {
	cmd := PostgresCommand()
	if cmd.Name != "postgres" {
		t.Errorf("Name = %q, want postgres", cmd.Name)
	}
	if !containsString(cmd.Aliases, "pg") {
		t.Errorf("Aliases = %v, want to include 'pg'", cmd.Aliases)
	}

	wantSubs := []string{
		"create",
		"list",
		"get",
		"status",
		"credentials",
		"connect",
		"proxy",
		"redeploy",
		"destroy",
		"users",
		"firewall",
		"storage-classes",
	}
	got := subNames(cmd.Subcommands)
	for _, want := range wantSubs {
		if !containsString(got, want) {
			t.Errorf("Subcommands missing %q (have %v)", want, got)
		}
	}
}

func TestValidatePostgresName(t *testing.T) {
	valid := []string{"app", "app-db", "pg1", "a"}
	for _, name := range valid {
		if err := validatePostgresName(name); err != nil {
			t.Errorf("validatePostgresName(%q) returned error: %v", name, err)
		}
	}

	invalid := []string{"", "-app", "app-", "App", "app_db", "a.b"}
	for _, name := range invalid {
		if err := validatePostgresName(name); err == nil {
			t.Errorf("validatePostgresName(%q) returned nil error", name)
		}
	}
}
