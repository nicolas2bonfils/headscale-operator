package rest

import (
	"context"
	"net/http"
	"testing"
)

func TestCreateAPIKey(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/apikey" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"apiKey":"hskey-api-secret"}`))
	})

	key, err := c.CreateAPIKey(context.Background(), nil)
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if key != "hskey-api-secret" {
		t.Fatalf("unexpected key: %q", key)
	}
}
