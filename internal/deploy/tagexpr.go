// Package deploy provides tag expression evaluation using Go's native AST parser.
// The expression syntax supports: key, key=value, & (AND), | (OR), and parentheses.
package deploy

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"1ctl/internal/api"
)

// EvaluateTagExpr evaluates a machine tag expression against a set of labels.
//
// Expression syntax:
//
//	key              → checks that satusky.com/<key> exists and is non-empty
//	key=value        → checks that satusky.com/<key> == value
//	expr & expr      → AND (both must match)
//	expr | expr      → OR (either must match)
//	(expr)           → grouping
//
// Label keys are automatically prefixed with satusky.com/ if not already prefixed.
func EvaluateTagExpr(expr string, labels map[string]string) (bool, error) {
	if expr == "" || labels == nil {
		return false, nil
	}

	// Sanitize to valid Go expression syntax.
	sanitized := strings.ReplaceAll(expr, "=", "==")
	sanitized = strings.ReplaceAll(sanitized, "|", "||")
	sanitized = strings.ReplaceAll(sanitized, "&", "&&")

	parsed, err := parser.ParseExpr(sanitized)
	if err != nil {
		return false, fmt.Errorf("invalid tag expression: %w", err)
	}

	return evalTagAST(parsed, labels), nil
}

func evalTagAST(node ast.Node, labels map[string]string) bool {
	switch n := node.(type) {
	case *ast.BinaryExpr:
		switch n.Op {
		case token.LAND: // &&
			return evalTagAST(n.X, labels) && evalTagAST(n.Y, labels)
		case token.LOR: // ||
			return evalTagAST(n.X, labels) || evalTagAST(n.Y, labels)
		case token.EQL: // ==
			return evalTagEq(n.X, n.Y, labels)
		}
		return false
	case *ast.ParenExpr:
		return evalTagAST(n.X, labels)
	case *ast.Ident:
	// Standalone identifier: value must be "true".
		key := normalizeTagKey(n.Name)
		val, ok := labels[key]
		return ok && val == "true"
	}
	return false
}

func evalTagEq(x, y ast.Expr, labels map[string]string) bool {
	keyIdent, okX := x.(*ast.Ident)
	valIdent, okY := y.(*ast.Ident)
	if !okX || !okY {
		return false
	}
	key := normalizeTagKey(keyIdent.Name)
	actual, ok := labels[key]
	if !ok {
		return false
	}
	return actual == valIdent.Name
}

func normalizeTagKey(key string) string {
	if strings.HasPrefix(key, "satusky.com/") {
		return key
	}
	return api.MachineTagLabelPrefix + key
}
