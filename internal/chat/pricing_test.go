package chat

import (
	"fmt"
	"math"
	"testing"
)

func TestRateFor(t *testing.T) {
	tests := []struct {
		provider  Provider
		wantOK    bool
		wantInput float64
	}{
		{provider: ProviderOpenAI, wantOK: true, wantInput: 0.15},
		{provider: ProviderClaude, wantOK: true, wantInput: 3.00},
		{provider: ProviderDeepSeek, wantOK: true, wantInput: 0.05},
		{provider: Provider("unknown"), wantOK: false},
	}
	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			rate, ok := RateFor(tt.provider)
			if ok != tt.wantOK {
				t.Fatalf("RateFor(%q) ok = %v, want %v", tt.provider, ok, tt.wantOK)
			}
			if ok && math.Abs(rate.InputUSD-tt.wantInput) > 1e-9 {
				t.Errorf("RateFor(%q) InputUSD = %v, want %v", tt.provider, rate.InputUSD, tt.wantInput)
			}
		})
	}
}

func TestRateCost(t *testing.T) {
	rate := Rate{InputUSD: 0.15, OutputUSD: 0.60}
	tests := []struct {
		name         string
		promptTokens int
		totalTokens  int
		want         float64
	}{
		{name: "zero tokens", want: 0},
		{name: "prompt only", promptTokens: 1_000_000, totalTokens: 1_000_000, want: 0.15},
		{name: "completion only", promptTokens: 0, totalTokens: 1_000_000, want: 0.60},
		{name: "mixed", promptTokens: 1_000, totalTokens: 1_204, want: 0.00015 + 0.0001224},
		{name: "negative completion clamped", promptTokens: 500, totalTokens: 100, want: 0.000075},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rate.Cost(tt.promptTokens, tt.totalTokens)
			if math.Abs(got-tt.want) > 1e-12 {
				t.Errorf("Cost(%d, %d) = %v, want %v", tt.promptTokens, tt.totalTokens, got, tt.want)
			}
		})
	}
}

func TestRateCostRoundingForFooter(t *testing.T) {
	// The footer formats with %.4f: 1204 tokens on openai ≈ $0.0003.
	rate, ok := RateFor(ProviderOpenAI)
	if !ok {
		t.Fatal("RateFor(openai) not found")
	}
	cost := rate.Cost(1000, 1204)
	formatted := fmt.Sprintf("%.4f", cost)
	if formatted != "0.0003" {
		t.Errorf("formatted cost = %q, want %q", formatted, "0.0003")
	}
}
