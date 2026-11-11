package postgres

import (
	"encoding/json"
	"testing"

	"1ctl/internal/api"
	"1ctl/internal/utils"
)

func lineValue(lines []utils.KeyValue, key string) (string, bool) {
	for _, line := range lines {
		if line.Key == key {
			return line.Value, true
		}
	}
	return "", false
}

func TestPostgresStatusLinesShowsExternalAccessError(t *testing.T) {
	status := &api.PostgresStatus{
		Status:              "degraded",
		ExternalAccessPhase: "error",
		ExternalAccessError: "endpointslices is forbidden",
	}
	lines := postgresStatusLines(status)

	if got, ok := lineValue(lines, "External access phase"); !ok || got != "error" {
		t.Errorf("External access phase = %q (present %v), want %q", got, ok, "error")
	}
	if got, ok := lineValue(lines, "External access error"); !ok || got != "endpointslices is forbidden" {
		t.Errorf("External access error = %q (present %v), want the backend error", got, ok)
	}
}

func TestPostgresStatusLinesHidesEmptyExternalAccessError(t *testing.T) {
	status := &api.PostgresStatus{Status: "ready", ExternalAccessible: true, ExternalAccessPhase: "ready"}
	lines := postgresStatusLines(status)

	if got, ok := lineValue(lines, "External access phase"); !ok || got != "ready" {
		t.Errorf("External access phase = %q (present %v), want %q", got, ok, "ready")
	}
	if _, ok := lineValue(lines, "External access error"); ok {
		t.Error("External access error line shown for an empty error")
	}
}

func TestPostgresStatusDecodesExternalAccessFields(t *testing.T) {
	body := `{"status":"degraded","external_accessible":false,"external_access_phase":"error","external_access_error":"forbidden"}`
	var status api.PostgresStatus
	if err := json.Unmarshal([]byte(body), &status); err != nil {
		t.Fatal(err)
	}
	if status.ExternalAccessPhase != "error" || status.ExternalAccessError != "forbidden" {
		t.Errorf("decoded phase=%q error=%q, want error/forbidden", status.ExternalAccessPhase, status.ExternalAccessError)
	}
}
