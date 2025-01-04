package utils

import (
	"bytes"
	"os"
	"testing"
)

func TestPrinter(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		args     []interface{}
		function string
		want     string
	}{
		{
			name:     "success message",
			format:   "Operation %s completed",
			args:     []interface{}{"test"},
			function: "Success",
			want:     "✓ Operation test completed\n",
		},
		{
			name:     "error message",
			format:   "Error: %s",
			args:     []interface{}{"failed"},
			function: "Error",
			want:     "✗ Error: failed\n",
		},
		{
			name:     "warning message",
			format:   "Warning: %s",
			args:     []interface{}{"caution"},
			function: "Warning",
			want:     "! Warning: caution\n",
		},
		{
			name:     "info message",
			format:   "Info: %s",
			args:     []interface{}{"notice"},
			function: "Info",
			want:     "ℹ Info: notice\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			var buf bytes.Buffer
			p := NewPrinter(&buf)

			// Call the appropriate function
			switch tt.function {
			case "Success":
				p.Success(tt.format, tt.args...)
			case "Error":
				p.Error(tt.format, tt.args...)
			case "Warning":
				p.Warning(tt.format, tt.args...)
			case "Info":
				p.Info(tt.format, tt.args...)
			}

			// Check output
			if got := buf.String(); got != tt.want {
				t.Errorf("%s() = %q, want %q", tt.function, got, tt.want)
			}
		})
	}
}

func TestNewPrinter(t *testing.T) {
	t.Run("with nil writer", func(t *testing.T) {
		p := NewPrinter(nil)
		if p.out != os.Stdout {
			t.Error("NewPrinter(nil) should use os.Stdout")
		}
	})

	t.Run("with custom writer", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewPrinter(&buf)
		if p.out != &buf {
			t.Error("NewPrinter should use provided writer")
		}
	})
}
