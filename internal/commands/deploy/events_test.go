package deploy

import (
	"testing"
	"time"

	"1ctl/internal/api"
)

func testEvent(offset int, category, eventType string) api.DeploymentEvent {
	return api.DeploymentEvent{
		At:       time.Date(2026, 8, 13, 10, 0, offset, 0, time.UTC),
		Category: category,
		Type:     eventType,
		Message:  category + " happened",
	}
}

// --follow re-polls a timeline derived from durable rows, so every poll returns
// the events it has already printed. Without a stable key each poll would
// reprint the whole history.
func TestEventKeyIsStableAcrossPolls(t *testing.T) {
	first := testEvent(5, "dns", "verified")
	second := testEvent(5, "dns", "verified")
	if first.Key() != second.Key() {
		t.Fatalf("Key() = %q and %q, want identical keys for the same event", first.Key(), second.Key())
	}
}

// Two events at the same instant in different categories are different events;
// collapsing them would silently drop one from the stream.
func TestEventKeyDistinguishesConcurrentEvents(t *testing.T) {
	dns := testEvent(5, "dns", "verified")
	reconcile := testEvent(5, "reconcile", "completed")
	if dns.Key() == reconcile.Key() {
		t.Fatalf("Key() collided for different categories at the same instant: %q", dns.Key())
	}
}

func TestTrimToLastKeepsTheMostRecent(t *testing.T) {
	events := []api.DeploymentEvent{
		testEvent(1, "deployment", "accepted"),
		testEvent(2, "reconcile", "queued"),
		testEvent(3, "reconcile", "completed"),
	}

	trimmed := trimToLast(events, 2)

	if len(trimmed) != 2 {
		t.Fatalf("len = %d, want 2", len(trimmed))
	}
	if trimmed[0].Type != "queued" || trimmed[1].Type != "completed" {
		t.Fatalf("trimmed = %v, want the two most recent events", eventTypesOf(trimmed))
	}
}

func TestTrimToLastReturnsEverythingWhenUnset(t *testing.T) {
	events := []api.DeploymentEvent{testEvent(1, "deployment", "accepted"), testEvent(2, "dns", "pending")}

	for _, last := range []int{0, -1, 5} {
		if got := trimToLast(events, last); len(got) != len(events) {
			t.Fatalf("trimToLast(events, %d) len = %d, want %d", last, len(got), len(events))
		}
	}
}

// The detail line must surface the reason first. An error buried behind the
// expected target is the confusion this command exists to remove.
func TestEventDetailPrefersTheActionableField(t *testing.T) {
	generation := int64(4)
	event := api.DeploymentEvent{
		Category: "reconcile", Type: "failed", Level: "warning",
		Detail:     map[string]string{"fqdn": "x.satusky.com", "error": "admission denied", "attempts": "3"},
		Generation: &generation,
	}

	detail := formatEventDetail(event)

	if want := "(error=admission denied gen=4)"; detail != want {
		t.Fatalf("formatEventDetail() = %q, want %q", detail, want)
	}
}

func TestEventDetailIsEmptyWithoutUsefulFields(t *testing.T) {
	event := api.DeploymentEvent{Category: "reconcile", Type: "queued", Detail: map[string]string{"operation": "apply"}}

	if detail := formatEventDetail(event); detail != "" {
		t.Fatalf("formatEventDetail() = %q, want empty", detail)
	}
}

func eventTypesOf(events []api.DeploymentEvent) []string {
	types := make([]string, 0, len(events))
	for _, event := range events {
		types = append(types, event.Type)
	}
	return types
}

// The source rows are mutable: re-completing a reconciliation task rewrites
// completed_at. Keying on the timestamp made one event reprint on every poll
// with a new time, so a single deployment looked like it completed repeatedly.
func TestEventKeyIgnoresAMovingTimestamp(t *testing.T) {
	first := api.DeploymentEvent{
		ID: "task-1:completed", At: time.Date(2026, 8, 13, 12, 14, 19, 0, time.UTC),
		Category: "reconcile", Type: "completed",
	}
	moved := first
	moved.At = time.Date(2026, 8, 13, 12, 16, 6, 0, time.UTC)

	if first.Key() != moved.Key() {
		t.Fatalf("Key() changed when only the timestamp moved: %q vs %q", first.Key(), moved.Key())
	}
}

// Distinct source rows stay distinct, so a genuine second reconciliation is
// still reported.
func TestEventKeyDistinguishesDifferentSourceRows(t *testing.T) {
	first := api.DeploymentEvent{ID: "task-1:completed", Category: "reconcile", Type: "completed"}
	second := api.DeploymentEvent{ID: "task-2:completed", Category: "reconcile", Type: "completed"}

	if first.Key() == second.Key() {
		t.Fatal("Key() collided across different reconciliation tasks")
	}
}

// An older server sends no ID; the timestamp fallback must still work.
func TestEventKeyFallsBackWithoutAnID(t *testing.T) {
	event := api.DeploymentEvent{At: time.Date(2026, 8, 13, 12, 14, 19, 0, time.UTC), Category: "dns", Type: "verified"}

	if event.Key() == "" {
		t.Fatal("Key() = empty for an event with no ID")
	}
}
