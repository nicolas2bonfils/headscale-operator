package rest

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"
)

// APIKey is a Headscale API key, without its secret value.
type APIKey struct {
	ID         string     `json:"id"`
	Prefix     string     `json:"prefix"`
	CreatedAt  *time.Time `json:"createdAt,omitempty"`
	Expiration *time.Time `json:"expiration,omitempty"`
	LastSeen   *time.Time `json:"lastSeen,omitempty"`
}

// CreateAPIKey creates an API key and returns its secret value. The secret is
// only available once.
func (c *Client) CreateAPIKey(ctx context.Context, expiration *time.Time) (string, error) {
	body := map[string]any{}
	if expiration != nil && !expiration.IsZero() {
		body["expiration"] = expiration.UTC().Format(time.RFC3339)
	}

	var out struct {
		APIKey string `json:"apiKey"`
	}
	if err := c.do(ctx, http.MethodPost, "/api/v1/apikey", nil, body, &out); err != nil {
		return "", err
	}
	if out.APIKey == "" {
		return "", errors.New("headscale api: create api key response has no key")
	}
	return out.APIKey, nil
}

// ListAPIKeys returns every API key, without secrets.
func (c *Client) ListAPIKeys(ctx context.Context) ([]APIKey, error) {
	var out struct {
		APIKeys []APIKey `json:"apiKeys"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/v1/apikey", nil, nil, &out); err != nil {
		return nil, err
	}
	return out.APIKeys, nil
}

// ExpireAPIKey expires the key with the given prefix.
func (c *Client) ExpireAPIKey(ctx context.Context, prefix string) error {
	if prefix == "" {
		return errors.New("api key prefix is required")
	}
	body := map[string]string{"prefix": prefix}
	return c.do(ctx, http.MethodPost, "/api/v1/apikey/expire", nil, body, nil)
}

// DeleteAPIKey deletes the key with the given prefix.
func (c *Client) DeleteAPIKey(ctx context.Context, prefix string) error {
	if prefix == "" {
		return errors.New("api key prefix is required")
	}
	q := url.Values{"prefix": {prefix}}
	return c.do(ctx, http.MethodDelete, "/api/v1/apikey/"+url.PathEscape(prefix), q, nil, nil)
}
