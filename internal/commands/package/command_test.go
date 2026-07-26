package packagecmd

import "testing"

func TestCreateCommandExposesChartFlag(t *testing.T) {
	for _, flag := range createCommand().Flags {
		if flag.Names()[0] == flagChart {
			return
		}
	}
	t.Fatal("package create does not expose --chart")
}
