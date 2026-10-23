package rest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestAuthAndCreateUser(t *testing.T) {
	var gotAuth, gotBody string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/user" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"user":{"id":"7","name":"alice","displayName":"Alice","email":"alice@example.com"}}`))
	})

	user, err := c.CreateUser(context.Background(), "alice", "Alice", "alice@example.com", "")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if user.ID != "7" || user.Name != "alice" {
		t.Fatalf("unexpected user: %+v", user)
	}
	if gotAuth != "Bearer test-key" {
		t.Fatalf("unexpected Authorization header: %q", gotAuth)
	}
	var body map[string]string
	if err := json.Unmarshal([]byte(gotBody), &body); err != nil {
		t.Fatalf("decoding request body: %v", err)
	}
	if body["name"] != "alice" || body["email"] != "alice@example.com" {
		t.Fatalf("unexpected request body: %s", gotBody)
	}
}

func TestGetUserByName(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("name") != "bob" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"users":[{"id":"9","name":"bob"}]}`))
	})

	user, err := c.GetUserByName(context.Background(), "bob")
	if err != nil {
		t.Fatalf("GetUserByName: %v", err)
	}
	if user.ID != "9" {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestGetUserByNameEmptyList(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"users":[]}`))
	})

	_, err := c.GetUserByName(context.Background(), "missing")
	if err != ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestGrpcStyleNotFound(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":5,"message":"user not found","details":[]}`))
	})

	_, err := c.GetUserByName(context.Background(), "missing")
	if err != ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestProblemJSONAlreadyExists(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"type":"https://headscale.net/errors/conflict","title":"Conflict","status":409,"detail":"user already exists"}`))
	})

	_, err := c.CreateUser(context.Background(), "alice", "", "", "")
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if !apiErr.IsAlreadyExists() {
		t.Fatalf("expected IsAlreadyExists, got %+v", apiErr)
	}
	if apiErr.Detail != "user already exists" {
		t.Fatalf("unexpected detail: %+v", apiErr)
	}
}
