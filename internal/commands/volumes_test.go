package commands

import "testing"

func TestVolumesCommandStructure(t *testing.T) {
	cmd := VolumesCommand()
	if cmd.Name != "volumes" {
		t.Errorf("Name = %q, want volumes", cmd.Name)
	}
	if !containsString(cmd.Aliases, "volume") {
		t.Errorf("Aliases = %v, want to include 'volume'", cmd.Aliases)
	}
	wantSubs := []string{"list", "inspect", "detach", "destroy"}
	got := subNames(cmd.Commands)
	for _, w := range wantSubs {
		if !containsString(got, w) {
			t.Errorf("Subcommands missing %q (have %v)", w, got)
		}
	}
}
