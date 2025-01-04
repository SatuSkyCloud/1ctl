package utils

import (
	"fmt"
	"io"
	"os"
	"strings"

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
	fmt.Fprintf(p.out, "%s\n", SuccessColor("✅ "+message))
}

// Error prints an error message
func (p *Printer) Error(format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	fmt.Fprintf(p.out, "%s\n", ErrorColor("❌ "+message))
}

// Warning prints a warning message
func (p *Printer) Warning(format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	fmt.Fprintf(p.out, "%s\n", WarnColor("❗️ "+message))
}

// Info prints an info message
func (p *Printer) Info(format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	fmt.Fprintf(p.out, "%s\n", InfoColor("💡 "+message))
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
