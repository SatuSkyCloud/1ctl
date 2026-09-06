package deploy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"1ctl/internal/api"
	"1ctl/internal/utils"
)

func TestDestroyJSONRequiresConfirmationAndEmitsOnlyOperation(t *testing.T) {
	const id = "3b521364-98b0-4787-bd07-a311bff3f223"
	deletes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/v1/deployments/"+id {
			deletes++
			w.WriteHeader(http.StatusAccepted)
			_, _ = io.WriteString(w, `{"data":{"deployment_id":"`+id+`","status":"requested","terminal":false}}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	setupDeploymentStatusTest(t, server.URL)
	output := captureDeletionOutput(t, "json", func() {
		err := handleDestroyDeployment(context.Background(), DestroyInput{DeploymentID: id, RetainVolumes: true, NoWait: true})
		if err == nil || !strings.Contains(err.Error(), "--yes") {
			t.Fatalf("expected explicit confirmation error, got %v", err)
		}
	})
	if output != "" || deletes != 0 {
		t.Fatalf("unconfirmed deletion wrote output or mutated: %q, %d", output, deletes)
	}
	output = captureDeletionOutput(t, "json", func() {
		if err := handleDestroyDeployment(context.Background(), DestroyInput{DeploymentID: id, RetainVolumes: true, NoWait: true, Yes: true}); err != nil {
			t.Fatal(err)
		}
	})
	var operation api.DeploymentDeletionOperation
	if err := json.Unmarshal([]byte(output), &operation); err != nil {
		t.Fatalf("output must be a single JSON document: %v; %q", err, output)
	}
	if deletes != 1 || operation.Status != "requested" || operation.Terminal {
		t.Fatalf("accepted deletion was misreported: %+v, calls=%d", operation, deletes)
	}
}

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
