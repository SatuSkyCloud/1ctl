package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"1ctl/internal/utils"
)

// consoleTokenURL is where users create API tokens in the control panel.
const consoleTokenURL = "https://cloud.satusky.com/<org-id>/token" // #nosec G101 -- console URL template, not a credential

// Stable error codes for local auth diagnostics (raised before any request).
const (
	codeTokenRequired    = "AUTH_TOKEN_REQUIRED"      // #nosec G101 -- error code name, not a credential
	codeTokenMalformed   = "AUTH_TOKEN_MALFORMED"     // #nosec G101 -- error code name, not a credential
	codeTokenExpired     = "AUTH_TOKEN_EXPIRED"       // #nosec G101 -- error code name, not a credential
	codeTokenWrongKind   = "AUTH_TOKEN_NOT_API_TOKEN" // #nosec G101 -- error code name, not a credential
	codeTokenRejected    = "AUTH_TOKEN_REJECTED"      // #nosec G101 -- error code name, not a credential
	tokenKindAPIToken    = "api_token"
	tokenKindSession     = "session"
	jwtSegmentCount      = 3
	bearerPrefixLowerLen = len("bearer ")
)

// createTokenSteps is the remediation shown whenever the user needs a token.
func createTokenSteps() []string {
	return []string{
		fmt.Sprintf("Open the SatuSky console > API Tokens (%s) and create a token", consoleTokenURL),
		"Run: 1ctl auth login --token <token>   (or: export SATUSKY_API_KEY=<token> && 1ctl auth login)",
		"Browser / email-password login is not available in the CLI yet",
	}
}

// errTokenRequired is returned when no token was given and none is saved.
func errTokenRequired() error {
	return utils.NewLocalDiagnosticError(
		"no API token given: 1ctl logs in with an API token from the SatuSky console",
		codeTokenRequired,
		createTokenSteps(),
	)
}

// normalizeToken removes copy/paste noise that makes a valid token fail the
// server's signature check: surrounding whitespace, quotes, angle brackets
// (from "--token=<token>" examples) and an "Authorization: Bearer" prefix.
func normalizeToken(raw string) string {
	token := strings.TrimSpace(raw)
	if len(token) > bearerPrefixLowerLen && strings.EqualFold(token[:bearerPrefixLowerLen], "bearer ") {
		token = strings.TrimSpace(token[bearerPrefixLowerLen:])
	}
	for _, pair := range [][2]string{{`"`, `"`}, {`'`, `'`}, {"<", ">"}, {"`", "`"}} {
		if len(token) >= 2 && strings.HasPrefix(token, pair[0]) && strings.HasSuffix(token, pair[1]) {
			token = strings.TrimSpace(token[1 : len(token)-1])
		}
	}
	return token
}

// tokenClaims is the subset of unverified JWT claims the CLI inspects to give
// a precise error before contacting the API. The signature is NOT verified
// here; the server stays the only authority on validity.
type tokenClaims struct {
	Exp       *float64 `json:"exp"`
	TokenKind string   `json:"token_kind"`
}

// precheckToken catches tokens that the server would reject with a generic
// "Invalid token format", and explains what is wrong instead.
func precheckToken(token string, now time.Time) error {
	parts := strings.Split(token, ".")
	if len(parts) != jwtSegmentCount || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return utils.NewLocalDiagnosticError(
			"this does not look like a SatuSky API token (expected a JWT: three base64 parts separated by dots)",
			codeTokenMalformed,
			append([]string{"Copy the whole token value shown once when the token is created; do not include quotes or spaces"}, createTokenSteps()...),
		)
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return malformedPayloadError()
	}
	var claims tokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return malformedPayloadError()
	}
	if claims.Exp != nil {
		expiresAt := time.Unix(int64(*claims.Exp), 0)
		if now.After(expiresAt) {
			return utils.NewLocalDiagnosticError(
				fmt.Sprintf("this API token expired on %s", expiresAt.UTC().Format(time.RFC3339)),
				codeTokenExpired,
				createTokenSteps(),
			)
		}
	}
	if claims.TokenKind == tokenKindSession {
		return utils.NewLocalDiagnosticError(
			"this is a browser session token, not an API token",
			codeTokenWrongKind,
			createTokenSteps(),
		)
	}
	return nil
}

func malformedPayloadError() error {
	return utils.NewLocalDiagnosticError(
		"this token is damaged: its payload cannot be decoded (was it copied completely?)",
		codeTokenMalformed,
		createTokenSteps(),
	)
}

// rejectedTokenRemediation explains the usual reasons a well-formed token is
// refused with 401, including the API URL the CLI talked to so a dev/prod
// mismatch is visible.
func rejectedTokenRemediation(apiURL string, fromSavedLogin bool) []string {
	steps := []string{}
	if fromSavedLogin {
		steps = append(steps, "The token saved by an earlier login is no longer accepted; log in again with a new token")
	}
	steps = append(steps,
		fmt.Sprintf("1ctl sent the token to %s: a token works only against the environment (prod/dev) whose console created it; check --profile, --api-url and SATUSKY_API_URL", apiURL),
		"Tokens are invalidated when revoked, expired, or after a platform signing-key rotation; create a new token if yours is older",
	)
	return append(steps, createTokenSteps()...)
}
