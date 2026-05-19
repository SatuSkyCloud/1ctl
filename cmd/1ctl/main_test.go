package main

import (
	"reflect"
	"testing"
)

func TestNormalizeGlobalOutputArgsMovesTrailingShortOutput(t *testing.T) {
	got := normalizeGlobalOutputArgs([]string{"1ctl", "deploy", "list", "-o", "json"})
	want := []string{"1ctl", "-o", "json", "deploy", "list"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeGlobalOutputArgs() = %v, want %v", got, want)
	}
}

func TestNormalizeGlobalOutputArgsMovesTrailingLongOutput(t *testing.T) {
	got := normalizeGlobalOutputArgs([]string{"1ctl", "domains", "check", "app.example.com", "--output=json"})
	want := []string{"1ctl", "--output=json", "domains", "check", "app.example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeGlobalOutputArgs() = %v, want %v", got, want)
	}
}

func TestNormalizeGlobalOutputArgsLeavesArgsWithoutOutput(t *testing.T) {
	args := []string{"1ctl", "deploy", "list"}
	got := normalizeGlobalOutputArgs(args)
	if !reflect.DeepEqual(got, args) {
		t.Fatalf("normalizeGlobalOutputArgs() = %v, want %v", got, args)
	}
}
