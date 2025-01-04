package version

import (
	"testing"
)

func TestGetVersionInfo(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		commit      string
		date        string
		wantVersion string
	}{
		{
			name:        "full version info",
			version:     "v1.0.0",
			commit:      "abc123",
			date:        "2024-01-01",
			wantVersion: "v1.0.0 (commit: abc123) (built: 2024-01-01)",
		},
		{
			name:        "development version",
			version:     "dev",
			commit:      "abc123",
			date:        "2024-01-01",
			wantVersion: "dev (commit: abc123) (built: 2024-01-01)",
		},
		{
			name:        "missing commit and date",
			version:     "v1.0.0",
			commit:      "",
			date:        "",
			wantVersion: "v1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			originalVersion := Version
			originalCommit := CommitHash
			originalDate := BuildDate
			defer func() {
				Version = originalVersion
				CommitHash = originalCommit
				BuildDate = originalDate
			}()

			// Set test values
			Version = tt.version
			CommitHash = tt.commit
			BuildDate = tt.date

			got := GetVersionInfo()
			if got != tt.wantVersion {
				t.Errorf("GetVersionInfo() = %v, want %v", got, tt.wantVersion)
			}
		})
	}
}
