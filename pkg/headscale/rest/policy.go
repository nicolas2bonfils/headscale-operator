package rest

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// Policy is the policy document stored by Headscale.
type Policy struct {
	Policy    string     `json:"policy"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// GetPolicy returns the stored policy.
func (c *Client) GetPolicy(ctx context.Context) (*Policy, error) {
	var out Policy
	if err := c.do(ctx, http.MethodGet, "/api/v1/policy", nil, nil, &out); err != nil {
		if isNotFound(err) {
			return nil, ErrPolicyNotSet
		}
		return nil, err
	}
	return &out, nil
}

// SetPolicy replaces the stored policy.
func (c *Client) SetPolicy(ctx context.Context, policy string) (*Policy, error) {
	body := map[string]string{"policy": policy}
	var out Policy
	if err := c.do(ctx, http.MethodPut, "/api/v1/policy", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CheckPolicy asks Headscale to validate a policy without storing it.
func (c *Client) CheckPolicy(ctx context.Context, policy string) error {
	if policy == "" {
		return errors.New("policy is required")
	}
	body := map[string]string{"policy": policy}
	return c.do(ctx, http.MethodPost, "/api/v1/policy/check", nil, body, nil)
}
