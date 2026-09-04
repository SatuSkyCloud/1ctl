package chat

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// spinnerFrames are the braille animation frames. Terminals in 2026
// handle these fine — the codebase already renders ▸ ⏎ ✓ ✅.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner renders an inline animated progress line to a terminal: a
// rotating frame plus fixed text, redrawn in place with \r and a
// clear-line escape. It is a strict no-op when disabled (non-TTY output,
// nil writer, tests) — every method returns without writing a byte, so
// piping and scripting output stays clean.
type Spinner struct {
	out      io.Writer
	enabled  bool
	frames   []string
	text     string
	interval time.Duration
	// manual drives frames synchronously via Step() instead of a timer
	// goroutine; tests set it to get deterministic animation.
	manual bool

	stop    chan struct{}
	done    chan struct{}
	mu      sync.Mutex
	started bool
	idx     int

	// showElapsed appends a " · 12s" counter to the animation so a
	// long-running operation never looks frozen. Only in timer mode.
	showElapsed bool
	startTime   time.Time
	now         func() time.Time // injectable clock (tests)
}

// NewSpinner builds a spinner for out. It animates only when out is a
// terminal (stdoutIsTTY), mirroring the ANSI-colour gating used across
// the REPL.
func NewSpinner(out io.Writer, text string) *Spinner {
	return NewSpinnerEnabled(out, text, stdoutIsTTY(out))
}

// NewSpinnerEnabled builds a spinner with an explicit enabled flag, so
// tests can force the animation on with a plain buffer.
func NewSpinnerEnabled(out io.Writer, text string, enabled bool) *Spinner {
	return &Spinner{
		out:      out,
		enabled:  enabled && out != nil,
		frames:   spinnerFrames,
		text:     text,
		interval: 80 * time.Millisecond,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
		now:      time.Now,
	}
}

// Elapsed enables the elapsed-seconds counter on the animation and
// returns the spinner for chaining.
func (s *Spinner) Elapsed() *Spinner {
	if s != nil {
		s.showElapsed = true
	}
	return s
}

// Start launches the animation goroutine (or arms manual mode). It is
// idempotent and a no-op when disabled.
func (s *Spinner) Start() {
	if s == nil || !s.enabled {
		return
	}
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	if s.now != nil {
		s.startTime = s.now()
	}
	s.mu.Unlock()
	if !s.manual {
		go s.run()
	}
}

// run drives the animation loop until Stop: on each tick it redraws the
// line (\r + clear-line escape + frame + text).
func (s *Spinner) run() {
	defer close(s.done)
	t := time.NewTicker(s.interval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			s.mu.Lock()
			if !s.started {
				s.mu.Unlock()
				return
			}
			s.drawLocked(s.renderLocked())
			s.idx++
			s.mu.Unlock()
		case <-s.stop:
			return
		}
	}
}

// Step draws the next frame synchronously. Only meaningful in manual
// mode (tests); a no-op otherwise, including when disabled.
func (s *Spinner) Step() {
	if s == nil || !s.enabled || !s.manual {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		return
	}
	s.drawLocked(s.renderLocked())
	s.idx++
}

// renderLocked builds the current line: "frame text" plus a " · 12s"
// elapsed counter when showElapsed is on and the spinner runs on a timer
// (manual/test mode stays deterministic). Callers hold mu.
func (s *Spinner) renderLocked() string {
	line := s.frames[s.idx%len(s.frames)] + " " + s.text
	if s.showElapsed && !s.manual && s.now != nil {
		secs := int(s.now().Sub(s.startTime).Seconds())
		if secs > 0 {
			line += fmt.Sprintf(" · %ds", secs)
		}
	}
	return line
}

// Update changes the animation text. The next tick (or Step) draws the
// new text. A no-op when disabled.
func (s *Spinner) Update(text string) {
	if s == nil || !s.enabled {
		return
	}
	s.mu.Lock()
	s.text = text
	s.mu.Unlock()
}

// Stop halts the animation and clears the line. Idempotent; a no-op when
// disabled or never started.
func (s *Spinner) Stop() {
	if s == nil || !s.enabled {
		return
	}
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return
	}
	s.started = false
	if s.manual {
		s.mu.Unlock()
	} else {
		close(s.stop)
		s.mu.Unlock()
		<-s.done
	}
	s.mu.Lock()
	s.drawLocked("")
	s.mu.Unlock()
}

// StopWith halts the animation, clears the line, and writes a final line
// followed by a newline. A no-op when disabled.
func (s *Spinner) StopWith(line string) {
	if s == nil || !s.enabled {
		return
	}
	s.Stop()
	s.mu.Lock()
	_, _ = io.WriteString(s.out, line+"\n") //nolint:errcheck // best-effort output
	s.mu.Unlock()
}

// drawLocked redraws the current line in place. Callers hold mu.
func (s *Spinner) drawLocked(line string) {
	_, _ = io.WriteString(s.out, "\r\x1b[2K"+line) //nolint:errcheck // best-effort output
}
