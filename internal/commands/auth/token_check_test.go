package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
)

// isolateStore points the CLI context at an empty temp dir so tests never
// read or write the developer's real ~/.satusky profiles.
func isolateStore(t *testing.T) {
	t.Helper()
	original := satuskyctx.Default()
	satuskyctx.SetDefault(satuskyctx.NewTestStore(t.TempDir()))
	t.Cleanup(func() { satuskyctx.SetDefault(original) })
	t.Setenv("SATUSKY_API_KEY", "")
}

func fakeJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	return header + "." + base64.RawURLEncoding.EncodeToString(body) + ".c2ln"
}

func diagCode(err error) string {
	var d *utils.LocalDiagnosticError
	if errors.As(err, &d) {
		return d.Code
	}
	return ""
}

func TestNormalizeToken(t *testing.T) {
	cases := map[string]string{
		"a.b.c":              "a.b.c",
		"  a.b.c\n":          "a.b.c",
		`"a.b.c"`:            "a.b.c",
		"'a.b.c'":            "a.b.c",
		"<a.b.c>":            "a.b.c",
		"Bearer a.b.c":       "a.b.c",
		"bearer  a.b.c ":     "a.b.c",
		"\"Bearer a.b.c\"\n": "Bearer a.b.c",
		"":                   "",
	}
	for in, want := range cases {
		if got := normalizeToken(in); got != want {
			t.Errorf("normalizeToken(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPrecheckToken(t *testing.T) {
	now := time.Unix(2_000_000_000, 0)
	future := float64(now.Add(time.Hour).Unix())
	past := float64(now.Add(-time.Hour).Unix())

	cases := []struct {
		name  string
		token string
		want  string
	}{
		{"valid api token", fakeJWT(t, map[string]any{"exp": future, "token_kind": "api_token"}), ""},
		{"legacy token without kind", fakeJWT(t, map[string]any{"exp": future}), ""},
		{"not a jwt", "sk_live_abcdef", codeTokenMalformed},
		{"empty segment", "a..c", codeTokenMalformed},
		{"bad payload", "a.!!!.c", codeTokenMalformed},
		{"expired", fakeJWT(t, map[string]any{"exp": past, "token_kind": "api_token"}), codeTokenExpired},
		{"session token", fakeJWT(t, map[string]any{"exp": future, "token_kind": "session"}), codeTokenWrongKind},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := precheckToken(tc.token, now)
			if got := diagCode(err); got != tc.want {
				t.Fatalf("code = %q (err %v), want %q", got, err, tc.want)
			}
		})
	}
}

func TestHandleLoginWithoutTokenExplainsHowToGetOne(t *testing.T) {
	isolateStore(t)
	err := handleLogin(context.Background(), authLoginInput{})
	if diagCode(err) != codeTokenRequired {
		t.Fatalf("want %s, got %v", codeTokenRequired, err)
	}
	var d *utils.LocalDiagnosticError
	errors.As(err, &d)
	if !strings.Contains(strings.Join(d.Remediation, "\n"), "API Tokens") {
		t.Fatalf("remediation should point at the console API Tokens page: %v", d.Remediation)
	}
}

func TestHandleLogin401AddsRemediation(t *testing.T) {
	isolateStore(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.ContainsAny(body["token"], " \"<>\n") {
			t.Errorf("token was not normalized before sending: %q", body["token"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":"AUTHENTICATION_REQUIRED","error":true,"message":"Invalid token format","valid":false}`))
	}))
	defer srv.Close()
	t.Setenv("SATUSKY_API_URL", srv.URL)

	token := fakeJWT(t, map[string]any{"exp": float64(time.Now().Add(time.Hour).Unix()), "token_kind": "api_token"})
	err := handleLogin(context.Background(), authLoginInput{Token: " <" + token + ">\n"})

	var statusErr *api.HTTPStatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("want *api.HTTPStatusError, got %T: %v", err, err)
	}
	if statusErr.StatusCode != http.StatusUnauthorized || statusErr.Code != "AUTHENTICATION_REQUIRED" {
		t.Fatalf("unexpected status/code: %d %s", statusErr.StatusCode, statusErr.Code)
	}
	if !strings.Contains(statusErr.Message, "Invalid token format") {
		t.Fatalf("message should keep the server message, got %q", statusErr.Message)
	}
	joined := strings.Join(statusErr.Remediation, "\n")
	if !strings.Contains(joined, srv.URL) || !strings.Contains(joined, "API Tokens") {
		t.Fatalf("remediation should name the API URL and the console page: %v", statusErr.Remediation)
	}
}
