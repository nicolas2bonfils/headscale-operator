package headscale

import (
	"context"
	"time"

	"github.com/infradohq/headscale-operator/pkg/headscale/rest"
)

// API is the subset of the Headscale API used by the controllers.
//
// The operator is migrating from the gRPC client in this package to the REST
// client in pkg/headscale/rest, because Headscale 0.30 removes gRPC. The
// interface is the migration seam: controllers should depend on API, not on a
// concrete client, so the gRPC implementation can be deleted in one step.
type API interface {
	Close() error
	Version(ctx context.Context) (string, error)

	CreateUser(ctx context.Context, username, displayName, email, pictureURL string) (*rest.User, error)
	GetUserByName(ctx context.Context, username string) (*rest.User, error)
	DeleteUser(ctx context.Context, id string) error

	CreatePreAuthKey(ctx context.Context, userID string, reusable, ephemeral bool, expiration *time.Time, tags []string) (*rest.PreAuthKey, error)
	DeletePreAuthKey(ctx context.Context, id string) error

	SetPolicy(ctx context.Context, policy string) (*rest.Policy, error)
	GetPolicy(ctx context.Context) (*rest.Policy, error)
}

var _ API = (*rest.Client)(nil)
