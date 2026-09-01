package chat

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// newTestSpinner builds a manual-mode spinner: frames advance
// synchronously via Step(), so tests are deterministic and race-free.
func newTestSpinner(enabled bool) (*Spinner, *bytes.Buffer) {
	var buf bytes.Buffer
	sp := NewSpinnerEnabled(&buf, "working", enabled)
	sp.manual = true
	return sp, &buf
}

func TestSpinnerStartWritesFrames(t *testing.T) {
	sp, buf := newTestSpinner(true)
	sp.Start()
	sp.Step()
	sp.Step()
	got := buf.String()
	if !strings.Contains(got, spinnerFrames[0]+" working") {
		t.Errorf("first frame missing: %q", got)
	}
	if !strings.Contains(got, spinnerFrames[1]+" working") {
		t.Errorf("second frame missing: %q", got)
	}
	if !strings.HasPrefix(got, "\r\x1b[2K") {
		t.Errorf("frames must start with carriage return + clear-line: %q", got)
	}
	sp.Stop()
}

func TestSpinnerStopClearsLine(t *testing.T) {
	sp, buf := newTestSpinner(true)
	sp.Start()
	sp.Step()
	sp.Stop()
	got := buf.String()
	// The final write must be a bare clear-line (no trailing frame text).
	if !strings.HasSuffix(got, "\r\x1b[2K") {
		t.Errorf("buffer must end with a bare clear-line: %q", got)
	}
	if strings.HasSuffix(got, spinnerFrames[0]+" working") {
		t.Errorf("frame text left after Stop: %q", got)
	}
}

func TestSpinnerStopWithWritesFinalLine(t *testing.T) {
	sp, buf := newTestSpinner(true)
	sp.Start()
	sp.Step()
	sp.StopWith("✓ done")
	got := buf.String()
	if !strings.HasSuffix(got, "\r\x1b[2K✓ done\n") {
		t.Errorf("StopWith must clear then write the final line: %q", got)
	}
}

func TestSpinnerUpdateChangesText(t *testing.T) {
	sp, buf := newTestSpinner(true)
	sp.Start()
	sp.Update("second task")
	sp.Step()
	if !strings.Contains(buf.String(), spinnerFrames[0]+" second task") {
		t.Errorf("updated text not drawn: %q", buf.String())
	}
	sp.Stop()
}

func TestSpinnerDisabledWritesNothing(t *testing.T) {
	sp, buf := newTestSpinner(false)
	sp.Start()
	sp.Step()
	sp.Update("nope")
	sp.StopWith("nope")
	sp.Stop()
	if buf.Len() != 0 {
		t.Errorf("disabled spinner wrote %d bytes: %q", buf.Len(), buf.String())
	}
}

func TestSpinnerNilWriterAndNilReceiver(t *testing.T) {
	sp := NewSpinnerEnabled(nil, "x", true) // enabled but nil writer: must be disabled
	sp.Start()
	sp.Step()
	sp.Stop()
	sp.StopWith("x")
	var nilSp *Spinner
	nilSp.Start()
	nilSp.Step()
	nilSp.Stop()
	nilSp.StopWith("x")
	nilSp.Update("x")
}

func TestSpinnerStopTwiceIdempotent(t *testing.T) {
	sp, buf := newTestSpinner(true)
	sp.Start()
	sp.Step()
	sp.Stop()
	before := buf.Len()
	sp.Stop()
	if buf.Len() != before {
		t.Errorf("second Stop wrote more output")
	}
}

// TestSpinnerElapsedRendering verifies the elapsed-seconds counter that
// keeps long-running operations from looking frozen. It drives renderLocked
// directly with a fixed clock so the test is deterministic.
func TestSpinnerElapsedRendering(t *testing.T) {
	var buf bytes.Buffer
	fixedNow := time.Unix(1_700_000_000, 0)
	sp := NewSpinnerEnabled(&buf, "running…", true)
	sp.showElapsed = true
	sp.now = func() time.Time { return fixedNow }
	sp.idx = 0
	sp.text = "running…"
	sp.manual = false // timer mode: elapsed counter is active

	// Before any time has passed: no counter yet.
	sp.startTime = fixedNow
	sp.started = true
	line := sp.renderLocked()
	if strings.Contains(line, "·") {
		t.Errorf("renderLocked before elapsed = %q, want no counter", line)
	}

	// After 12s: counter appears.
	sp.startTime = fixedNow.Add(-12 * time.Second)
	line = sp.renderLocked()
	if !strings.Contains(line, "· 12s") {
		t.Errorf("renderLocked after 12s = %q, want '· 12s'", line)
	}

	// Manual mode stays deterministic (no counter) — tests rely on it.
	sp.manual = true
	sp.startTime = fixedNow.Add(-30 * time.Second)
	if line := sp.renderLocked(); strings.Contains(line, "· 30s") {
		t.Errorf("manual-mode renderLocked = %q, want no counter", line)
	}
}

// TestSpinnerElapsedDisabled ensures the counter is off by default.
func TestSpinnerElapsedDisabled(t *testing.T) {
	var buf bytes.Buffer
	sp := NewSpinnerEnabled(&buf, "x", true)
	sp.manual = true
	sp.started = true
	sp.now = func() time.Time { return time.Unix(1_700_000_000, 0) }
	sp.startTime = sp.now().Add(-5 * time.Second)
	if line := sp.renderLocked(); strings.Contains(line, "·") {
		t.Errorf("default renderLocked = %q, want no counter", line)
	}
}
