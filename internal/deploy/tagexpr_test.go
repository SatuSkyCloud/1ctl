package deploy

import "testing"

func TestEvaluateTagExpr(t *testing.T) {
	labels := map[string]string{
		"satusky.com/tier":       "compute",
		"satusky.com/production": "true",
		"satusky.com/cat":        "true",
		"satusky.com/compute":    "true",
	}

	tests := []struct {
		name string
		expr string
		want bool
	}{
		{"bare ident true", "production", true},
		{"bare ident no match", "nonexistent", false},
		{"bare ident not true value", "tier", false}, // tier=compute, not tier=true
		{"value match", "tier=compute", true},
		{"value no match", "tier=staging", false},
		{"AND both match", "tier=compute&production", true},
		{"AND one missing", "tier=compute&nonexistent", false},
		{"OR both match", "tier=compute|cat", true},
		{"OR one matches", "tier=staging|cat", true},
		{"OR none match", "tier=staging|nonexistent", false},
		{"grouped AND", "(tier=compute|tier=staging)&production", true},
		{"grouped AND no match", "(tier=compute|tier=staging)&nonexistent", false},
		{"nested groups", "((tier=compute|tier=staging)&production)", true},
		{"empty expr", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateTagExpr(tt.expr, labels)
			if err != nil {
				t.Errorf("EvaluateTagExpr(%q) error: %v", tt.expr, err)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateTagExpr(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}


