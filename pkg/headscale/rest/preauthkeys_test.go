package rest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestCreatePreAuthKeyRequest(t *testing.T) {
	expiration := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	var body map[string]any
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(b, &body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		_, _ = w.Write([]byte(`{"preAuthKey":{"id":"3","key":"hskey-auth-abc","reusable":true,"ephemeral":false,"aclTags":["tag:ci"]}}`))
	})

	key, err := c.CreatePreAuthKey(context.Background(), "7", true, false, &expiration, []string{"tag:ci"})
	if err != nil {
		t.Fatalf("CreatePreAuthKey: %v", err)
	}
	if key.ID != "3" || key.Key != "hskey-auth-abc" {
		t.Fatalf("unexpected key: %+v", key)
	}
	if body["user"] != "7" {
		t.Fatalf("user should be a string, got %#v", body["user"])
	}
	if body["expiration"] != "2026-10-24T12:00:00Z" {
		t.Fatalf("unexpected expiration: %#v", body["expiration"])
	}
	tags, ok := body["aclTags"].([]any)
	if !ok || len(tags) != 1 || tags[0] != "tag:ci" {
		t.Fatalf("unexpected tags: %#v", body["aclTags"])
	}
}

func TestDeletePreAuthKeyQuery(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.URL.Query().Get("id") != "42" {
			t.Errorf("unexpected id query: %s", r.URL.RawQuery)
		}
	})

	if err := c.DeletePreAuthKey(context.Background(), "42"); err != nil {
		t.Fatalf("DeletePreAuthKey: %v", err)
	}
}
