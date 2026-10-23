package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// User is a Headscale user.
type User struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	CreatedAt     *time.Time `json:"createdAt,omitempty"`
	DisplayName   string     `json:"displayName,omitempty"`
	Email         string     `json:"email,omitempty"`
	ProviderID    string     `json:"providerId,omitempty"`
	Provider      string     `json:"provider,omitempty"`
	ProfilePicURL string     `json:"profilePicUrl,omitempty"`
}

// Uint64ID returns the numeric user ID.
func (u *User) Uint64ID() (uint64, error) {
	return ParseID(u.ID)
}

// ParseID parses a Headscale ID, which the API encodes as a string.
func ParseID(s string) (uint64, error) {
	if s == "" {
		return 0, errors.New("empty id")
	}
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing id %q: %w", s, err)
	}
	return id, nil
}

// CreateUser creates a user and returns it.
func (c *Client) CreateUser(ctx context.Context, username, displayName, email, pictureURL string) (*User, error) {
	body := map[string]string{"name": username}
	if displayName != "" {
		body["displayName"] = displayName
	}
	if email != "" {
		body["email"] = email
	}
	if pictureURL != "" {
		body["pictureUrl"] = pictureURL
	}

	var out struct {
		User User `json:"user"`
	}
	if err := c.do(ctx, http.MethodPost, "/api/v1/user", nil, body, &out); err != nil {
		return nil, err
	}
	if out.User.ID == "" {
		return nil, errors.New("headscale api: create user response has no user")
	}
	return &out.User, nil
}

// GetUserByName returns the user with the given name.
func (c *Client) GetUserByName(ctx context.Context, username string) (*User, error) {
	var out struct {
		Users []User `json:"users"`
	}
	q := url.Values{"name": {username}}
	if err := c.do(ctx, http.MethodGet, "/api/v1/user", q, nil, &out); err != nil {
		if isNotFound(err) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if len(out.Users) == 0 {
		return nil, ErrUserNotFound
	}
	return &out.Users[0], nil
}

// ListUsers returns all users.
func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	var out struct {
		Users []User `json:"users"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/v1/user", nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Users, nil
}

// DeleteUser deletes the user with the given ID.
func (c *Client) DeleteUser(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("user id is required")
	}
	err := c.do(ctx, http.MethodDelete, "/api/v1/user/"+url.PathEscape(id), nil, nil, nil)
	if isNotFound(err) {
		return ErrUserNotFound
	}
	return err
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrUserNotFound) || errors.Is(err, ErrPreAuthKeyNotFound) {
		return true
	}
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.IsNotFound()
}
