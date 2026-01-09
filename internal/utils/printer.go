package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

// Printer handles formatted output
type Printer struct {
	out io.Writer
}

var (
	// Color functions
	InfoColor    = color.New(color.FgCyan).SprintfFunc()
	SuccessColor = color.New(color.FgGreen).SprintfFunc()
	WarnColor    = color.New(color.FgYellow).SprintfFunc()
	ErrorColor   = color.New(color.FgRed).SprintfFunc()
	BoldColor    = color.New(color.Bold).SprintfFunc()
	DividerColor = color.New(color.FgCyan).SprintfFunc()
)

// Add these new functions
var loadingChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// NewPrinter creates a new printer instance
func NewPrinter(out io.Writer) *Printer {
	if out == nil {
		out = os.Stdout
	}
	return &Printer{out: out}
}

// Global printer instance for convenience
var defaultPrinter = NewPrinter(os.Stdout)

// Success prints a success message
func (p *Printer) Success(format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	_, _ = fmt.Fprintf(p.out, "%s\n", SuccessColor("✅ "+message)) //nolint:errcheck
}

// Error prints an error message
func (p *Printer) Error(format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	_, _ = fmt.Fprintf(p.out, "%s\n", ErrorColor("❌ "+message)) //nolint:errcheck
}

// Warning prints a warning message
func (p *Printer) Warning(format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	_, _ = fmt.Fprintf(p.out, "%s\n", WarnColor("❗️ "+message)) //nolint:errcheck
}

// Info prints an info message
func (p *Printer) Info(format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	_, _ = fmt.Fprintf(p.out, "%s\n", InfoColor("💡 "+message)) //nolint:errcheck
}

// Global functions that use the default printer
func PrintSuccess(format string, a ...interface{}) {
	defaultPrinter.Success(format, a...)
}

func PrintError(format string, a ...interface{}) {
	defaultPrinter.Error(format, a...)
}

func PrintWarning(format string, a ...interface{}) {
	defaultPrinter.Warning(format, a...)
}

func PrintInfo(format string, a ...interface{}) {
	defaultPrinter.Info(format, a...)
}

// PrintStep prints a step in the deployment process
func PrintStep(step, total int, message, resource string) {
	fmt.Printf("%s: %s %s\n",
		InfoColor(fmt.Sprintf("Step %d/%d", step, total)),
		SuccessColor(message),
		WarnColor(resource))
}

// PrintHeader prints a header with bold text and underline
func PrintHeader(format string, a ...interface{}) {
	text := fmt.Sprintf(format, a...)
	fmt.Println(BoldColor(text))
	underline := strings.Repeat("─", len(text))
	fmt.Println(DividerColor(underline))
}

// PrintStatusLine prints a status line with label and value
func PrintStatusLine(label, value string) {
	fmt.Printf("%s: %s\n", BoldColor(label), value)
}

// PrintDivider prints a divider line
func PrintDivider() {
	fmt.Println(DividerColor("---"))
}

// PrintLoadingStep prints a step in the deployment process with a loading animation
func PrintLoadingStep(step, total int, message, resource string, done bool) {
	if done {
		fmt.Printf("\r%s: %s %s %s\n",
			InfoColor(fmt.Sprintf("Step %d/%d", step, total)),
			SuccessColor(message),
			WarnColor(resource),
			SuccessColor("✓"))
	} else {
		spinner := loadingChars[step%len(loadingChars)]
		fmt.Printf("\r%s: %s %s %s",
			InfoColor(fmt.Sprintf("Step %d/%d", step, total)),
			SuccessColor(message),
			WarnColor(resource),
			InfoColor(spinner))
	}
}

// ============================================================================
// Table Output
// ============================================================================

// TableConfig holds configuration for table printing
type TableConfig struct {
	Headers    []string
	MinWidths  []int // Minimum column widths
	MaxWidth   int   // Max total width (0 = unlimited)
	Padding    int   // Padding between columns
	NoHeader   bool  // Skip header row
	HeaderLine bool  // Print line under header
}

// DefaultTableConfig returns default table configuration
func DefaultTableConfig() TableConfig {
	return TableConfig{
		Padding:    2,
		HeaderLine: true,
	}
}

// PrintTable prints data in a formatted table
func PrintTable(headers []string, rows [][]string) {
	config := DefaultTableConfig()
	config.Headers = headers
	PrintTableWithConfig(rows, config)
}

// PrintTableWithConfig prints data in a formatted table with custom config
func PrintTableWithConfig(rows [][]string, config TableConfig) {
	if len(config.Headers) == 0 && len(rows) == 0 {
		return
	}

	// Calculate column widths
	numCols := len(config.Headers)
	if numCols == 0 && len(rows) > 0 {
		numCols = len(rows[0])
	}

	colWidths := make([]int, numCols)

	// Consider header widths
	for i, h := range config.Headers {
		if i < numCols && len(h) > colWidths[i] {
			colWidths[i] = len(h)
		}
	}

	// Consider row widths
	for _, row := range rows {
		for i, cell := range row {
			if i < numCols && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Apply minimum widths
	for i, minW := range config.MinWidths {
		if i < numCols && minW > colWidths[i] {
			colWidths[i] = minW
		}
	}

	// Print header
	if !config.NoHeader && len(config.Headers) > 0 {
		headerParts := make([]string, numCols)
		for i := 0; i < numCols; i++ {
			header := ""
			if i < len(config.Headers) {
				header = config.Headers[i]
			}
			headerParts[i] = BoldColor(padRight(header, colWidths[i]))
		}
		fmt.Println(strings.Join(headerParts, strings.Repeat(" ", config.Padding)))

		// Print header underline
		if config.HeaderLine {
			lineParts := make([]string, numCols)
			for i := 0; i < numCols; i++ {
				lineParts[i] = DividerColor(strings.Repeat("─", colWidths[i]))
			}
			fmt.Println(strings.Join(lineParts, strings.Repeat(" ", config.Padding)))
		}
	}

	// Print rows
	for _, row := range rows {
		rowParts := make([]string, numCols)
		for i := 0; i < numCols; i++ {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			rowParts[i] = padRight(cell, colWidths[i])
		}
		fmt.Println(strings.Join(rowParts, strings.Repeat(" ", config.Padding)))
	}
}

// padRight pads a string to the right with spaces
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// ============================================================================
// Progress Bar
// ============================================================================

// PrintProgressBar prints a progress bar
func PrintProgressBar(current, total int, label string) {
	if total <= 0 {
		total = 1
	}
	percent := float64(current) / float64(total) * 100
	if percent > 100 {
		percent = 100
	}

	barWidth := 30
	filled := int(float64(barWidth) * percent / 100)
	empty := barWidth - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	fmt.Printf("\r%s %s %s %.0f%%",
		InfoColor(label),
		SuccessColor(bar),
		BoldColor(fmt.Sprintf("%d/%d", current, total)),
		percent)

	if current >= total {
		fmt.Println()
	}
}

// PrintProgressBarWithSize prints a progress bar with byte sizes
func PrintProgressBarWithSize(current, total int64, label string) {
	if total <= 0 {
		total = 1
	}
	percent := float64(current) / float64(total) * 100
	if percent > 100 {
		percent = 100
	}

	barWidth := 25
	filled := int(float64(barWidth) * percent / 100)
	empty := barWidth - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	fmt.Printf("\r%s %s %s/%s %.0f%%",
		InfoColor(label),
		SuccessColor(bar),
		FormatBytes(current),
		FormatBytes(total),
		percent)

	if current >= total {
		fmt.Println()
	}
}

// ============================================================================
// JSON Output
// ============================================================================

// PrintJSON prints data as formatted JSON
func PrintJSON(data interface{}) error {
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

// PrintJSONCompact prints data as compact JSON
func PrintJSONCompact(data interface{}) error {
	output, err := json.Marshal(data)
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

// ============================================================================
// Key-Value Output
// ============================================================================

// KeyValue represents a key-value pair for display
type KeyValue struct {
	Key   string
	Value string
}

// PrintKeyValue prints aligned key-value pairs
func PrintKeyValue(pairs []KeyValue) {
	PrintKeyValueWithIndent(pairs, 0)
}

// PrintKeyValueWithIndent prints aligned key-value pairs with indentation
func PrintKeyValueWithIndent(pairs []KeyValue, indent int) {
	if len(pairs) == 0 {
		return
	}

	// Find max key width
	maxKeyWidth := 0
	for _, kv := range pairs {
		if len(kv.Key) > maxKeyWidth {
			maxKeyWidth = len(kv.Key)
		}
	}

	indentStr := strings.Repeat(" ", indent)
	for _, kv := range pairs {
		fmt.Printf("%s%s: %s\n",
			indentStr,
			BoldColor(padRight(kv.Key, maxKeyWidth)),
			kv.Value)
	}
}

// ============================================================================
// Formatting Helpers
// ============================================================================

// FormatBytes formats bytes into human-readable format
func FormatBytes(bytes int64) string {
	if bytes < 0 {
		return "0 B"
	}

	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	if bytes == 0 {
		return "0 B"
	}

	exp := int(math.Log(float64(bytes)) / math.Log(1024))
	if exp >= len(units) {
		exp = len(units) - 1
	}

	value := float64(bytes) / math.Pow(1024, float64(exp))

	if value >= 100 || exp == 0 {
		return fmt.Sprintf("%.0f %s", value, units[exp])
	} else if value >= 10 {
		return fmt.Sprintf("%.1f %s", value, units[exp])
	}
	return fmt.Sprintf("%.2f %s", value, units[exp])
}

// FormatBytesKubernetes formats bytes like Kubernetes (Ki, Mi, Gi)
func FormatBytesKubernetes(bytes int64) string {
	if bytes < 0 {
		return "0"
	}

	units := []string{"", "Ki", "Mi", "Gi", "Ti", "Pi"}
	if bytes == 0 {
		return "0"
	}

	exp := int(math.Log(float64(bytes)) / math.Log(1024))
	if exp >= len(units) {
		exp = len(units) - 1
	}

	value := float64(bytes) / math.Pow(1024, float64(exp))

	if value >= 100 || exp == 0 {
		return fmt.Sprintf("%.0f%s", value, units[exp])
	} else if value >= 10 {
		return fmt.Sprintf("%.1f%s", value, units[exp])
	}
	return fmt.Sprintf("%.2f%s", value, units[exp])
}

// FormatDuration formats a duration into human-readable format
func FormatDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}

	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		if secs > 0 {
			return fmt.Sprintf("%dm %ds", mins, secs)
		}
		return fmt.Sprintf("%dm", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		mins := int(d.Minutes()) % 60
		if mins > 0 {
			return fmt.Sprintf("%dh %dm", hours, mins)
		}
		return fmt.Sprintf("%dh", hours)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	if hours > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	return fmt.Sprintf("%dd", days)
}

// FormatTimeAgo formats a time as relative (e.g., "2 hours ago")
func FormatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "N/A"
	}

	d := time.Since(t)
	if d < 0 {
		return "just now"
	}

	if d < time.Minute {
		secs := int(d.Seconds())
		if secs <= 1 {
			return "just now"
		}
		return fmt.Sprintf("%d seconds ago", secs)
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	if d < 30*24*time.Hour {
		days := int(d.Hours()) / 24
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
	if d < 365*24*time.Hour {
		months := int(d.Hours()) / (24 * 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
	years := int(d.Hours()) / (24 * 365)
	if years == 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}

// ============================================================================
// Additional Helpers
// ============================================================================

// PrintBanner prints a banner header
func PrintBanner(text string) {
	width := len(text) + 4
	border := strings.Repeat("═", width)
	fmt.Println(DividerColor("╔" + border + "╗"))
	fmt.Printf("%s  %s  %s\n", DividerColor("║"), BoldColor(text), DividerColor("║"))
	fmt.Println(DividerColor("╚" + border + "╝"))
}

// PrintSection prints a section header
func PrintSection(title string) {
	fmt.Println()
	fmt.Println(BoldColor(title))
	fmt.Println(DividerColor(strings.Repeat("─", len(title))))
}

// PrintList prints a list of items with bullets
func PrintList(items []string) {
	for _, item := range items {
		fmt.Printf("  • %s\n", item)
	}
}

// PrintNumberedList prints a numbered list of items
func PrintNumberedList(items []string) {
	for i, item := range items {
		fmt.Printf("  %d. %s\n", i+1, item)
	}
}

// titleCase capitalizes the first letter of a string
func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// PrintStatusBadge prints a colored status badge
func PrintStatusBadge(status string) string {
	status = strings.ToLower(status)
	switch status {
	case "running", "active", "healthy", "ready", "completed", "success", "paid":
		return SuccessColor("● " + titleCase(status))
	case "pending", "waiting", "starting", "building", "processing":
		return WarnColor("● " + titleCase(status))
	case "failed", "error", "unhealthy", "crashed", "terminated":
		return ErrorColor("● " + titleCase(status))
	case "stopped", "paused", "inactive", "disabled":
		return InfoColor("○ " + titleCase(status))
	default:
		return status
	}
}

// PrintConfirmation prints a confirmation prompt message
func PrintConfirmation(message string) {
	fmt.Printf("%s %s ", WarnColor("?"), message)
}

// PrintUsageBar prints a usage bar (e.g., for storage/memory)
func PrintUsageBar(used, total int64, label string) {
	if total <= 0 {
		fmt.Printf("%s: N/A\n", label)
		return
	}

	percent := float64(used) / float64(total) * 100
	if percent > 100 {
		percent = 100
	}

	barWidth := 20
	filled := int(float64(barWidth) * percent / 100)
	empty := barWidth - filled

	var bar string
	if percent >= 90 {
		bar = ErrorColor(strings.Repeat("█", filled)) + strings.Repeat("░", empty)
	} else if percent >= 70 {
		bar = WarnColor(strings.Repeat("█", filled)) + strings.Repeat("░", empty)
	} else {
		bar = SuccessColor(strings.Repeat("█", filled)) + strings.Repeat("░", empty)
	}

	fmt.Printf("%s: %s %s/%s (%.0f%%)\n",
		BoldColor(label),
		bar,
		FormatBytes(used),
		FormatBytes(total),
		percent)
}
