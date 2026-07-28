package deploy

import (
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
