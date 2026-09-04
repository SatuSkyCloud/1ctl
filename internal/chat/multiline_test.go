package chat

import "testing"

func TestCompleteMultiline(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  bool
	}{
		{name: "empty", lines: nil, want: true},
		{name: "single line", lines: []string{"hello"}, want: true},
		{name: "backslash end", lines: []string{"foo \\"}, want: false},
		{name: "backslash end with trailing space", lines: []string{"foo \\  "}, want: false},
		{name: "backslash closed by next line", lines: []string{"foo \\", "bar"}, want: true},
		{name: "backslash on middle line", lines: []string{"a \\", "b", "c"}, want: true},
		{name: "balanced braces single line", lines: []string{"func() { return 1 }"}, want: true},
		{name: "unbalanced opening brace", lines: []string{"func() {"}, want: false},
		{name: "unbalanced brace closed later", lines: []string{"func() {", "return 1", "}"}, want: true},
		{name: "unbalanced opening bracket", lines: []string{"list := ["}, want: false},
		{name: "unbalanced opening paren", lines: []string{"foo("}, want: false},
		{name: "closing exceeds opening", lines: []string{"}"}, want: true},
		{name: "unbalanced double quote", lines: []string{`say "hello`}, want: false},
		{name: "unbalanced quote closed later", lines: []string{`say "hello`, `world"`}, want: true},
		{name: "balanced quotes", lines: []string{`say "hello world"`}, want: true},
		{name: "balanced multiline code", lines: []string{"func main() {", `  fmt.Println("hi")`, "}"}, want: true},
		{name: "prose with parens", lines: []string{"see (the docs) for details"}, want: true},
		{name: "empty continuation line", lines: []string{"func() {", ""}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := completeMultiline(tt.lines); got != tt.want {
				t.Errorf("completeMultiline(%q) = %v, want %v", tt.lines, got, tt.want)
			}
		})
	}
}

func TestJoinMultiline(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  string
	}{
		{name: "plain", lines: []string{"hello"}, want: "hello"},
		{name: "joined with newlines", lines: []string{"a", "b", "c"}, want: "a\nb\nc"},
		{name: "backslash markers dropped", lines: []string{"a \\", "b \\", "c"}, want: "a\nb\nc"},
		{name: "backslash with trailing space", lines: []string{"a \\ ", "b"}, want: "a\nb"},
		{name: "trailing whitespace trimmed", lines: []string{"a  ", "b\t"}, want: "a\nb"},
		{name: "interior backslash kept", lines: []string{`C:\dir`, "next"}, want: "C:\\dir\nnext"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := joinMultiline(tt.lines); got != tt.want {
				t.Errorf("joinMultiline(%q) = %q, want %q", tt.lines, got, tt.want)
			}
		})
	}
}
