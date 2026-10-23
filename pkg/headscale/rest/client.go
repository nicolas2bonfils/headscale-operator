// Package rest is a small HTTP client for the Headscale v1 API (/api/v1).
//
// It targets both Headscale 0.29 (grpc-gateway, Swagger 2.0) and 0.30+
// (Huma, OpenAPI 3.1). The JSON resource shapes are the same across both; the
// error formats differ, which APIError hides.
package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultTimeout is the default HTTP client timeout.
	DefaultTimeout = 15 * time.Second

	// maxResponseBytes bounds how much of a response body is read.
	maxResponseBytes = 4 << 20
)

var (
	// ErrUserNotFound is returned when a user does not exist in Headscale.
	ErrUserNotFound = errors.New("user not found")

	// ErrPreAuthKeyNotFound is returned when a preauth key does not exist.
	ErrPreAuthKeyNotFound = errors.New("preauth key not found")

	// ErrPolicyNotSet is returned when the instance has no policy stored.
	ErrPolicyNotSet = errors.New("policy not set")
)

// APIError is a non-2xx response from the Headscale API.
type APIError struct {
	StatusCode int
	// Code is the gRPC status code name on 0.29 (e.g. "NotFound") or the
	// RFC 7807 "type" value on 0.30.
	Code    string
	Message string
	Detail  string
}

func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = http.StatusText(e.StatusCode)
	}
	if e.Detail != "" && e.Detail != msg {
		return fmt.Sprintf("headscale api: %s: %s", msg, e.Detail)
	}
	return fmt.Sprintf("headscale api: %s", msg)
}

// IsNotFound reports whether the error is a 404 or a gRPC NotFound.
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound || e.Code == "NotFound" || e.Code == "5"
}

// IsAlreadyExists reports whether the resource already exists.
func (e *APIError) IsAlreadyExists() bool {
	return e.StatusCode == http.StatusConflict || e.Code == "AlreadyExists" || e.Code == "6"
}

// Client talks to a Headscale instance over HTTP.
type Client struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *http.Client
	version    string
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient overrides the HTTP client (useful in tests).
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.httpClient = h }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.httpClient.Timeout = d }
}

// New creates a client for baseURL, for example http://headscale.ns.svc:8080
// or https://headscale.example.com. apiKey is sent as a Bearer token.
func New(baseURL, apiKey string, opts ...Option) (*Client, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("parsing base URL: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("base URL %q must include scheme and host", baseURL)
	}

	c := &Client{
		baseURL:    u,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: DefaultTimeout},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// Close implements the client lifecycle used by the controllers. The HTTP
// client keeps no persistent connection that needs closing.
func (c *Client) Close() error { return nil }

// Version returns the Headscale version reported by /version, caching it.
func (c *Client) Version(ctx context.Context) (string, error) {
	if c.version != "" {
		return c.version, nil
	}

	var raw json.RawMessage
	if err := c.do(ctx, http.MethodGet, "/version", nil, nil, &raw); err != nil {
		return "", err
	}

	var obj struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && obj.Version != "" {
		c.version = obj.Version
		return c.version, nil
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil && s != "" {
		c.version = s
		return c.version, nil
	}

	c.version = strings.TrimSpace(strings.Trim(string(raw), `"`))
	return c.version, nil
}

// Health calls the unauthenticated /health endpoint.
func (c *Client) Health(ctx context.Context) error {
	return c.do(ctx, http.MethodGet, "/health", nil, nil, nil)
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, in, out any) error {
	u := *c.baseURL
	u.Path = strings.TrimRight(c.baseURL.Path, "/") + path
	u.RawQuery = query.Encode()

	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("encoding %s %s request: %w", method, path, err)
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return fmt.Errorf("building %s %s request: %w", method, path, err)
	}
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return fmt.Errorf("reading %s %s response: %w", method, path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseAPIError(resp.StatusCode, data)
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decoding %s %s response: %w", method, path, err)
	}
	return nil
}

// parseAPIError understands both the grpc-gateway status object used by 0.29
// and the RFC 7807 problem+json document used by 0.30.
func parseAPIError(statusCode int, data []byte) error {
	apiErr := &APIError{StatusCode: statusCode}

	var problem struct {
		Type   string `json:"type"`
		Title  string `json:"title"`
		Detail string `json:"detail"`
		Status int    `json:"status"`
	}
	if err := json.Unmarshal(data, &problem); err == nil && (problem.Title != "" || problem.Type != "") {
		apiErr.Code = problem.Type
		apiErr.Message = problem.Title
		apiErr.Detail = problem.Detail
		return apiErr
	}

	var grpcErr struct {
		Code    any    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(data, &grpcErr); err == nil && grpcErr.Message != "" {
		apiErr.Code = grpcCodeName(grpcErr.Code)
		apiErr.Message = grpcErr.Message
		return apiErr
	}

	apiErr.Message = strings.TrimSpace(string(data))
	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(statusCode)
	}
	return apiErr
}

func grpcCodeName(code any) string {
	switch v := code.(type) {
	case string:
		return v
	case float64:
		return grpcCodeNameByNumber(int(v))
	default:
		return ""
	}
}

func grpcCodeNameByNumber(code int) string {
	switch code {
	case 3:
		return "InvalidArgument"
	case 5:
		return "NotFound"
	case 6:
		return "AlreadyExists"
	case 7:
		return "PermissionDenied"
	case 16:
		return "Unauthenticated"
	default:
		return strconv.Itoa(code)
	}
}
