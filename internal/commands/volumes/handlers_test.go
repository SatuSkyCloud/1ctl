package volumes

import (
	"strings"
	"testing"

	"1ctl/internal/api"
)

func TestDesiredAttachmentTextDistinguishesRequestedAndObservedState(t *testing.T) {
	status := &api.VolumeLifecycleStatus{
		Volume: api.Volume{DesiredAttached: false},
		Mount:  api.VolumeMountStatus{Attached: true, Mounted: true},
	}
	if got := desiredAttachmentText(status); !strings.Contains(got, "pending") {
		t.Fatalf("desiredAttachmentText() = %q, want pending detachment", got)
	}

	status.Mount = api.VolumeMountStatus{}
	if got := desiredAttachmentText(status); got != "detached" {
		t.Fatalf("desiredAttachmentText() = %q, want detached", got)
	}

	status.Volume.DesiredAttached = true
	if got := desiredAttachmentText(status); !strings.Contains(got, "pending") {
		t.Fatalf("desiredAttachmentText() = %q, want pending attachment", got)
	}
}
