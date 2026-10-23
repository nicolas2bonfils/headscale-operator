package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// PreAuthKey is a Headscale pre-authentication key.
type PreAuthKey struct {
	ID         string     `json:"id"`
	Key        string     `json:"key"`
	User       *User      `json:"user,omitempty"`
	Reusable   bool       `json:"reusable"`
	Ephemeral  bool       `json:"ephemeral"`
	Used       bool       `json:"used"`
	Expiration *time.Time `json:"expiration,omitempty"`
	CreatedAt  *time.Time `json:"createdAt,omitempty"`
	ACLTags    []string   `json:"aclTags,omitempty"`
}

// Uint64ID returns the numeric key ID.
func (k *PreAuthKey) Uint64ID() (uint64, error) {
	return ParseID(k.ID)
}

// CreatePreAuthKey creates a key for userID. Pass an empty userID for a
// tags-only key, and a nil expiration for a key that never expires.
func (c *Client) CreatePreAuthKey(
	ctx context.Context,
	userID string,
	reusable, ephemeral bool,
	expiration *time.Time,
	tags []string,
) (*PreAuthKey, error) {
	body := map[string]any{
		"reusable":  reusable,
		"ephemeral": ephemeral,
	}
	if userID != "" {
		body["user"] = userID
	}
	if expiration != nil && !expiration.IsZero() {
		body["expiration"] = expiration.UTC().Format(time.RFC3339)
	}
	if len(tags) > 0 {
		body["aclTags"] = tags
	}

	var out struct {
		PreAuthKey PreAuthKey `json:"preAuthKey"`
	}
	if err := c.do(ctx, http.MethodPost, "/api/v1/preauthkey", nil, body, &out); err != nil {
		return nil, err
	}
	if out.PreAuthKey.Key == "" {
		return nil, errors.New("headscale api: create preauth key response has no key")
	}
	return &out.PreAuthKey, nil
}

// ListPreAuthKeys returns every preauth key.
func (c *Client) ListPreAuthKeys(ctx context.Context) ([]PreAuthKey, error) {
	var out struct {
		PreAuthKeys []PreAuthKey `json:"preAuthKeys"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/v1/preauthkey", nil, nil, &out); err != nil {
		return nil, err
	}
	return out.PreAuthKeys, nil
}

// DeletePreAuthKey deletes the key with the given ID.
func (c *Client) DeletePreAuthKey(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("preauth key id is required")
	}
	q := url.Values{"id": {id}}
	err := c.do(ctx, http.MethodDelete, "/api/v1/preauthkey", q, nil, nil)
	if isNotFound(err) {
		return ErrPreAuthKeyNotFound
	}
	return err
}

// ExpirePreAuthKey expires the key with the given ID.
func (c *Client) ExpirePreAuthKey(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("preauth key id is required")
	}
	body := map[string]string{"id": id}
	err := c.do(ctx, http.MethodPost, "/api/v1/preauthkey/expire", nil, body, nil)
	if isNotFound(err) {
		return ErrPreAuthKeyNotFound
	}
	if err != nil {
		return fmt.Errorf("expiring preauth key %s: %w", id, err)
	}
	return nil
}
