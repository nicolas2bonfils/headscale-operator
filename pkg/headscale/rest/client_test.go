package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := New(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestVersion(t *testing.T) {
	for name, body := range map[string]string{
		"object": `{"version":"0.29.5"}`,
		"string": `"0.30.0"`,
	} {
		t.Run(name, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(body))
			})
			v, err := c.Version(context.Background())
			if err != nil {
				t.Fatalf("Version: %v", err)
			}
			if v == "" {
				t.Fatal("empty version")
			}
		})
	}
}
