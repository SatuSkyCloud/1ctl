package deploy

import (
	"testing"
	"time"

	"1ctl/internal/api"

	"github.com/google/uuid"
)

func listTestDeployments(t *testing.T) []api.Deployment {
	t.Helper()
	return []api.Deployment{
		{
			DeploymentID: uuid.MustParse("8ee0f590-0fa3-44b4-84f4-f8df338041a9"),
			AppLabel:     "im-test",
			Status:       "ready",
			CreatedAt:    time.Now().Add(-time.Hour),
		},
		{
			DeploymentID: uuid.MustParse("2ea2a869-d2fc-429e-b5a4-b594c6bec443"),
			AppLabel:     "pl-test",
			Hostnames:    []string{"node-a", "node-b"},
			Status:       "ready",
			CreatedAt:    time.Now().Add(-time.Hour),
		},
	}
}

func TestDeploymentListTableDefaultHasNoURLColumn(t *testing.T) {
	headers, rows := deploymentListTable(listTestDeployments(t), nil, false)

	want := []string{"NAME", "DEPLOYMENT ID", "HOSTNAMES", "STATUS", "CREATED"}
	if len(headers) != len(want) {
		t.Fatalf("headers = %v, want %v", headers, want)
	}
	for i, h := range want {
		if headers[i] != h {
			t.Fatalf("headers = %v, want %v", headers, want)
		}
	}
	for _, row := range rows {
		if len(row) != len(headers) {
			t.Fatalf("row %v has %d cells, want %d", row, len(row), len(headers))
		}
	}
	if rows[1][2] != "node-a, node-b" {
		t.Fatalf("HOSTNAMES = %q, want the joined machine list", rows[1][2])
	}
}

func TestDeploymentListTableWideAddsURLColumn(t *testing.T) {
	domains := map[string]string{
		"8ee0f590-0fa3-44b4-84f4-f8df338041a9": "https://sleepyzebra-ogg4s8m.satusky.com",
	}

	headers, rows := deploymentListTable(listTestDeployments(t), domains, true)

	if headers[2] != "URL" {
		t.Fatalf("headers = %v, want URL in position 2", headers)
	}
	for _, row := range rows {
		if len(row) != len(headers) {
			t.Fatalf("row %v has %d cells, want %d", row, len(row), len(headers))
		}
	}
	if rows[0][2] != "https://sleepyzebra-ogg4s8m.satusky.com" {
		t.Fatalf("URL = %q, want the ingress domain", rows[0][2])
	}
	// A deployment with no ingress still renders a cell, otherwise the row is
	// short and every column after it shifts left.
	if rows[1][2] != "-" {
		t.Fatalf("URL for a deployment with no ingress = %q, want %q", rows[1][2], "-")
	}
	// The URL column is added, not substituted for HOSTNAMES.
	if rows[1][3] != "node-a, node-b" {
		t.Fatalf("HOSTNAMES = %q, want the joined machine list", rows[1][3])
	}
}

// Wide mode with no domain map at all is the path taken when the ingress lookup
// fails: the listing must still print rather than the command erroring out.
func TestDeploymentListTableWideSurvivesMissingDomains(t *testing.T) {
	_, rows := deploymentListTable(listTestDeployments(t), nil, true)

	for _, row := range rows {
		if len(row) != 6 {
			t.Fatalf("row %v has %d cells, want 6", row, len(row))
		}
		if row[2] != "-" {
			t.Fatalf("URL = %q, want %q", row[2], "-")
		}
	}
}
