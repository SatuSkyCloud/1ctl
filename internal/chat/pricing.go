package chat

// Rate is a per-provider pricing estimate in USD per 1M tokens. Values are
// best-effort approximations of current list prices — they drift as models
// and pricing change, are only used for the informational cost footer after
// each answer, and are never billed.
type Rate struct {
	// InputUSD is the estimated USD per 1M input (prompt) tokens.
	InputUSD float64
	// OutputUSD is the estimated USD per 1M output (completion) tokens.
	OutputUSD float64
}

// Cost estimates the USD cost of a completion: prompt tokens at the input
// rate, the remainder (total − prompt) at the output rate. Negative
// completion counts (broken usage reports) are clamped to zero.
func (r Rate) Cost(promptTokens, totalTokens int) float64 {
	completion := totalTokens - promptTokens
	if completion < 0 {
		completion = 0
	}
	return float64(promptTokens)/1e6*r.InputUSD + float64(completion)/1e6*r.OutputUSD
}

// pricing maps each provider to the estimated rate for its default model
// (see provider.go). All values are estimates, not guaranteed invoices.
var pricing = map[Provider]Rate{
	// gpt-4o-mini: ~$0.15 / $0.60 per 1M tokens (estimate).
	ProviderOpenAI: {InputUSD: 0.15, OutputUSD: 0.60},
	// claude-sonnet-4-6: ~$3 / $15 per 1M tokens (estimate).
	ProviderClaude: {InputUSD: 3.00, OutputUSD: 15.00},
	// deepseek-v4-flash: ~$0.05 / $0.50 per 1M tokens (estimate).
	ProviderDeepSeek: {InputUSD: 0.05, OutputUSD: 0.50},
}

// RateFor returns the estimated rate for a provider's default model. The
// bool is false when the provider has no known rate (custom providers or
// unknown names), in which case the caller should omit the cost footer.
func RateFor(p Provider) (Rate, bool) {
	rate, ok := pricing[p]
	return rate, ok
}
