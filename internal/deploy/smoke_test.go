package deploy

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckPublicURLSmokeRequiresSuccessfulHTTPResponse(t *testing.T) {
	tests := []struct {
		name   string
		code   int
		strict bool
		ready  bool
	}{
		{name: "successful response", code: http.StatusOK, strict: true, ready: true},
		{name: "failed response", code: http.StatusServiceUnavailable, strict: true, ready: false},
		{name: "non-strict reachability can accept not found", code: http.StatusNotFound, strict: false, ready: true},
		{name: "strict verification rejects not found", code: http.StatusNotFound, strict: true, ready: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.code)
			}))
			defer server.Close()
			result := CheckPublicURLSmoke(server.URL, []string{"/health"}, tt.strict)
			if result.Ready != tt.ready {
				t.Fatalf("CheckPublicURLSmoke() = %+v, want ready=%v", result, tt.ready)
			}
		})
	}
}

// The smoke resolved DNS through a hardcoded 8.8.8.8, which is unreachable from
// many environments — a GitHub Actions runner among them — so the smoke failed
// there, VerifyApplication() was always false, and a healthy deployment ended at
// "timeout waiting for verified deployment readiness". The platform had the same
// bug (#605) and fixed it by using the system resolver with an override.
func TestSmokeResolverDefaultsToTheSystemResolver(t *testing.T) {
	t.Setenv("SMOKE_DNS_RESOLVER", "")

	if got := smokeResolver(); got != net.DefaultResolver {
		t.Fatalf("smokeResolver() = %#v, want net.DefaultResolver", got)
	}
}

func TestSmokeResolverHonoursAPinnedServer(t *testing.T) {
	for _, pin := range []string{"1.1.1.1", "1.1.1.1:53"} {
		t.Run(pin, func(t *testing.T) {
			t.Setenv("SMOKE_DNS_RESOLVER", pin)

			got := smokeResolver()
			if got == net.DefaultResolver {
				t.Fatal("a pinned resolver must not fall back to the system resolver")
			}
			if got.Dial == nil {
				t.Fatal("a pinned resolver needs a Dial")
			}
		})
	}
}
