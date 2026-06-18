package commands

import "testing"

func TestDoctorCommandStructure(t *testing.T) {
	cmd := DoctorCommand()
	if cmd.Name != "doctor" {
		t.Fatalf("Name = %q, want doctor", cmd.Name)
	}

	wantFlags := []string{"deployment-id", "config", "health-path", "smoke"}
	gotFlags := make(map[string]bool)
	for _, flag := range cmd.Flags {
		for _, name := range flag.Names() {
			gotFlags[name] = true
		}
	}
	for _, want := range wantFlags {
		if !gotFlags[want] {
			t.Errorf("doctor command missing flag %q", want)
		}
	}
}
