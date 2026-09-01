package chat

import "strings"

// maxContinuationLines caps how many continuation lines are read for a
// still-incomplete multiline input before it is sent as-is. It prevents a
// stray unbalanced brace or quote from trapping the user in continuation
// mode forever.
const maxContinuationLines = 50

// continuationPrompt is shown instead of the main prompt while reading
// continuation lines of a multiline input.
const continuationPrompt = "... "

// completeMultiline reports whether the accumulated input lines form a
// complete message: the last line must not end with a backslash, and braces
// ({ } [ ] ( )) and double quotes must be balanced across all lines. An
// empty input is complete.
func completeMultiline(lines []string) bool {
	if len(lines) == 0 {
		return true
	}
	if strings.HasSuffix(strings.TrimRight(lines[len(lines)-1], " \t"), `\`) {
		return false
	}
	open, close := braceBalance(lines)
	for i := range open {
		if open[i] > close[i] {
			return false
		}
	}
	quotes := 0
	for _, l := range lines {
		quotes += strings.Count(l, `"`)
	}
	return quotes%2 == 0
}

// braceBalance counts opening and closing brace runes across all lines:
// index 0 is curly braces, 1 square brackets, 2 parentheses.
func braceBalance(lines []string) (open, close [3]int) {
	for _, l := range lines {
		for i := 0; i < len(l); i++ {
			switch l[i] {
			case '{':
				open[0]++
			case '}':
				close[0]++
			case '[':
				open[1]++
			case ']':
				close[1]++
			case '(':
				open[2]++
			case ')':
				close[2]++
			}
		}
	}
	return open, close
}

// joinMultiline joins input lines with newlines, dropping the trailing
// backslash continuation marker (and surrounding trailing whitespace) from
// each line so the model never sees the editing affordance.
func joinMultiline(lines []string) string {
	var b strings.Builder
	for i, l := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		t := strings.TrimRight(l, " \t")
		if strings.HasSuffix(t, `\`) {
			t = strings.TrimRight(t[:len(t)-1], " \t")
		}
		b.WriteString(t)
	}
	return b.String()
}
