package rest

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPolicyRoundTrip(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/policy":
			b, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(b), `"policy"`) {
				t.Errorf("missing policy field: %s", b)
			}
			_, _ = w.Write([]byte(`{"policy":"{\"acls\":[]}","updatedAt":"2026-10-23T20:00:00Z"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/policy":
			_, _ = w.Write([]byte(`{"policy":"{\"acls\":[]}"}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	if _, err := c.SetPolicy(context.Background(), `{"acls":[]}`); err != nil {
		t.Fatalf("SetPolicy: %v", err)
	}
	policy, err := c.GetPolicy(context.Background())
	if err != nil {
		t.Fatalf("GetPolicy: %v", err)
	}
	if policy.Policy != `{"acls":[]}` {
		t.Fatalf("unexpected policy: %+v", policy)
	}
}
