package deploy

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"1ctl/internal/api"
	"1ctl/internal/utils"
)

func TestPrintDeploymentDeletionOperationReportsPendingAndFailedDurableStates(t *testing.T) {
	pending := captureDeletionOutput(t, "table", func() {
		printDeploymentDeletionOperation(&api.DeploymentDeletionOperation{
			OperationID: "op-pending", DeploymentID: "dep-1", Status: "requested", State: "requested", StatusURL: "/v1/deployments/id/dep-1",
		})
	})
	for _, want := range []string{"Operation ID", "op-pending", "Status", "requested", "State", "Terminal", "false"} {
		if !strings.Contains(pending, want) {
			t.Fatalf("pending output missing %q: %s", want, pending)
		}
	}

	failed := &api.DeploymentDeletionOperation{
		OperationID: "op-failed", DeploymentID: "dep-1", Status: "failed", State: "failed", Terminal: true,
		RemediationCode: "retained_volume", RemediationDetail: "Delete the retained PVC after copying its data.",
	}
	table := captureDeletionOutput(t, "table", func() { printDeploymentDeletionOperation(failed) })
	for _, want := range []string{"Status", "failed", "State", "Terminal", "true", "Remediation code", "retained_volume", "Delete the retained PVC"} {
		if !strings.Contains(table, want) {
			t.Fatalf("failed output missing %q: %s", want, table)
		}
	}
	if got := deletionLifecycleError(failed); got != "retained_volume: Delete the retained PVC after copying its data." {
		t.Fatalf("deletionLifecycleError() = %q", got)
	}
}

func TestPrintDeploymentDeletionOperationProjectsRetainedResourcesInTableAndJSON(t *testing.T) {
	operation := &api.DeploymentDeletionOperation{
		OperationID: "op-1", DeploymentID: "dep-1", Status: "completed", State: "completed", Terminal: true,
		RetainedResources: []api.DeploymentDeletionRetainedResource{{
			ResourceClass: "kubernetes", Kind: "PersistentVolumeClaim", Resource: "persistentvolumeclaims", Namespace: "tenant-a", Name: "data",
		}},
	}
	table := captureDeletionOutput(t, "table", func() { printDeploymentDeletionOperation(operation) })
	for _, want := range []string{"Retained Resources", "CLASS", "KIND", "RESOURCE", "NAMESPACE", "NAME", "kubernetes", "PersistentVolumeClaim", "persistentvolumeclaims", "tenant-a", "data"} {
		if !strings.Contains(table, want) {
			t.Fatalf("table output missing %q: %s", want, table)
		}
	}

	output := captureDeletionOutput(t, "json", func() { printDeploymentDeletionOperation(operation) })
	var projection map[string]any
	if err := json.Unmarshal([]byte(output), &projection); err != nil {
		t.Fatalf("deletion JSON invalid: %v\n%s", err, output)
	}
	if projection["operation_id"] != "op-1" || projection["state"] != "completed" || projection["status"] != "completed" || projection["terminal"] != true {
		t.Fatalf("durable JSON projection = %#v", projection)
	}
	retained, ok := projection["retained_resources"].([]any)
	if !ok || len(retained) != 1 {
		t.Fatalf("retained_resources = %#v", projection["retained_resources"])
	}
	resource := retained[0].(map[string]any)
	if resource["resource_class"] != "kubernetes" || resource["name"] != "data" {
		t.Fatalf("retained resource = %#v", resource)
	}
}

func captureDeletionOutput(t *testing.T, format string, fn func()) string {
	t.Helper()
	originalFormat := "table"
	if utils.IsJSONOutput() {
		originalFormat = "json"
	}
	utils.SetOutputFormat(format)
	t.Cleanup(func() { utils.SetOutputFormat(originalFormat) })

	originalStdout := os.Stdout
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = write
	fn()
	_ = write.Close()
	os.Stdout = originalStdout
	output, err := io.ReadAll(read)
	_ = read.Close()
	if err != nil {
		t.Fatal(err)
	}
	return string(output)
}
