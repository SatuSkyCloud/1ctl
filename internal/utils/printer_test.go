package utils

import (
	"bytes"
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero bytes", 0, "0 B"},
		{"negative bytes", -100, "0 B"},
		{"bytes", 500, "500 B"},
		{"kilobytes", 1024, "1.00 KB"},
		{"kilobytes with decimal", 1536, "1.50 KB"},
		{"megabytes", 1048576, "1.00 MB"},
		{"megabytes large", 10485760, "10.0 MB"},
		{"gigabytes", 1073741824, "1.00 GB"},
		{"gigabytes large", 107374182400, "100 GB"},
		{"terabytes", 1099511627776, "1.00 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFormatBytesKubernetes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero bytes", 0, "0"},
		{"negative bytes", -100, "0"},
		{"bytes", 500, "500"},
		{"kibibytes", 1024, "1.00Ki"},
		{"mebibytes", 1048576, "1.00Mi"},
		{"gibibytes", 1073741824, "1.00Gi"},
		{"tebibytes", 1099511627776, "1.00Ti"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytesKubernetes(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytesKubernetes(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"milliseconds", 500 * time.Millisecond, "500ms"},
		{"seconds", 5 * time.Second, "5.0s"},
		{"minutes only", 5 * time.Minute, "5m"},
		{"minutes and seconds", 5*time.Minute + 30*time.Second, "5m 30s"},
		{"hours only", 2 * time.Hour, "2h"},
		{"hours and minutes", 2*time.Hour + 30*time.Minute, "2h 30m"},
		{"days only", 48 * time.Hour, "2d"},
		{"days and hours", 50 * time.Hour, "2d 2h"},
		{"negative duration", -5 * time.Second, "5.0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestFormatTimeAgo(t *testing.T) {
	tests := []struct {
		name     string
		time     time.Time
		contains string // Use contains since exact seconds might vary
	}{
		{"zero time", time.Time{}, "N/A"},
		{"just now", time.Now().Add(-500 * time.Millisecond), "just now"},
		{"seconds ago", time.Now().Add(-30 * time.Second), "seconds ago"},
		{"1 minute ago", time.Now().Add(-1 * time.Minute), "1 minute ago"},
		{"minutes ago", time.Now().Add(-5 * time.Minute), "minutes ago"},
		{"1 hour ago", time.Now().Add(-1 * time.Hour), "1 hour ago"},
		{"hours ago", time.Now().Add(-5 * time.Hour), "hours ago"},
		{"1 day ago", time.Now().Add(-24 * time.Hour), "1 day ago"},
		{"days ago", time.Now().Add(-72 * time.Hour), "days ago"},
		{"1 month ago", time.Now().Add(-35 * 24 * time.Hour), "1 month ago"},
		{"months ago", time.Now().Add(-90 * 24 * time.Hour), "months ago"},
		{"1 year ago", time.Now().Add(-400 * 24 * time.Hour), "1 year ago"},
		{"years ago", time.Now().Add(-800 * 24 * time.Hour), "years ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTimeAgo(tt.time)
			if result != tt.contains && !containsString(result, tt.contains) {
				t.Errorf("FormatTimeAgo(%v) = %q, want to contain %q", tt.time, result, tt.contains)
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && s[len(s)-len(substr):] == substr))
}

func TestTitleCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"lowercase", "hello", "Hello"},
		{"uppercase", "HELLO", "HELLO"},
		{"mixed case", "hELLO", "HELLO"},
		{"single char", "a", "A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := titleCase(tt.input)
			if result != tt.expected {
				t.Errorf("titleCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPrintStatusBadge(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"running", "running"},
		{"active", "active"},
		{"healthy", "healthy"},
		{"pending", "pending"},
		{"waiting", "waiting"},
		{"failed", "failed"},
		{"error", "error"},
		{"stopped", "stopped"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PrintStatusBadge(tt.status)
			// Status badge should not be empty
			if len(result) == 0 {
				t.Errorf("PrintStatusBadge(%q) returned empty string", tt.status)
			}
		})
	}
}

func TestPrinter(t *testing.T) {
	t.Run("NewPrinter with nil", func(t *testing.T) {
		p := NewPrinter(nil)
		if p == nil {
			t.Error("NewPrinter(nil) returned nil")
		}
	})

	t.Run("NewPrinter with buffer", func(t *testing.T) {
		buf := &bytes.Buffer{}
		p := NewPrinter(buf)
		if p == nil {
			t.Error("NewPrinter(buf) returned nil")
		}

		p.Success("test message")
		if buf.Len() == 0 {
			t.Error("Success() did not write to buffer")
		}
	})

	t.Run("Error message", func(t *testing.T) {
		buf := &bytes.Buffer{}
		p := NewPrinter(buf)
		p.Error("error message")
		if buf.Len() == 0 {
			t.Error("Error() did not write to buffer")
		}
	})

	t.Run("Warning message", func(t *testing.T) {
		buf := &bytes.Buffer{}
		p := NewPrinter(buf)
		p.Warning("warning message")
		if buf.Len() == 0 {
			t.Error("Warning() did not write to buffer")
		}
	})

	t.Run("Info message", func(t *testing.T) {
		buf := &bytes.Buffer{}
		p := NewPrinter(buf)
		p.Info("info message")
		if buf.Len() == 0 {
			t.Error("Info() did not write to buffer")
		}
	})
}

func TestPrintTable(t *testing.T) {
	t.Run("empty table", func(t *testing.T) {
		// Should not panic
		PrintTable([]string{}, [][]string{})
	})

	t.Run("table with headers only", func(t *testing.T) {
		// Should not panic
		PrintTable([]string{"Name", "Value"}, [][]string{})
	})

	t.Run("table with data", func(t *testing.T) {
		// Should not panic
		PrintTable(
			[]string{"Name", "Value", "Status"},
			[][]string{
				{"item1", "100", "active"},
				{"item2", "200", "inactive"},
			},
		)
	})
}

func TestPrintJSON(t *testing.T) {
	t.Run("valid struct", func(t *testing.T) {
		data := struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}{
			Name:  "test",
			Value: 123,
		}
		err := PrintJSON(data)
		if err != nil {
			t.Errorf("PrintJSON() returned error: %v", err)
		}
	})

	t.Run("valid map", func(t *testing.T) {
		data := map[string]interface{}{
			"name":  "test",
			"value": 123,
		}
		err := PrintJSON(data)
		if err != nil {
			t.Errorf("PrintJSON() returned error: %v", err)
		}
	})
}

func TestPrintJSONCompact(t *testing.T) {
	t.Run("valid struct", func(t *testing.T) {
		data := struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}{
			Name:  "test",
			Value: 123,
		}
		err := PrintJSONCompact(data)
		if err != nil {
			t.Errorf("PrintJSONCompact() returned error: %v", err)
		}
	})
}

func TestPrintKeyValue(t *testing.T) {
	t.Run("empty pairs", func(t *testing.T) {
		// Should not panic
		PrintKeyValue([]KeyValue{})
	})

	t.Run("with pairs", func(t *testing.T) {
		// Should not panic
		PrintKeyValue([]KeyValue{
			{Key: "Name", Value: "Test"},
			{Key: "Status", Value: "Active"},
		})
	})
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		expected string
	}{
		{"no padding needed", "hello", 3, "hello"},
		{"exact width", "hello", 5, "hello"},
		{"padding needed", "hi", 5, "hi   "},
		{"empty string", "", 3, "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := padRight(tt.input, tt.width)
			if result != tt.expected {
				t.Errorf("padRight(%q, %d) = %q, want %q", tt.input, tt.width, result, tt.expected)
			}
		})
	}
}
