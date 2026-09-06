package deploy

import (
	"1ctl/internal/utils"
	"io"
	"os"
	"testing"
)

func TestDeploymentProgressDoesNotPolluteJSON(t *testing.T) {
	utils.SetOutputFormat("json")
	t.Cleanup(func() { utils.SetOutputFormat("table") })
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	original := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = original; writer.Close() }()
	printDeployInfo("Using pre-built image: %s", "fixture")
	progress := &deploymentProgress{step: 1, total: 2, message: "fixture"}
	progress.print()
	progress.complete()
	writer.Close()
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if len(output) != 0 {
		t.Fatalf("JSON stdout contains human progress: %q", output)
	}
}
