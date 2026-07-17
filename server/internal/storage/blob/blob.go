// Package blob abstracts the object store.
// Implementations: MinIO/S3 (cloud) and local filesystem (selfhost).
package blob

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/johnnycube/openbeehive-app/server/internal/config"
)

type Store interface {
	// Put stores an object and returns its key.
	Put(ctx context.Context, key, contentType string, r io.Reader, size int64) error
	// Get returns the object for reading.
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	// PresignedPut creates a time-limited upload URL (direct upload
	// from the browser, without load on the app server).
	PresignedPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, error)
	// PublicURL for read access.
	PublicURL(key string) string
}

func New(cfg *config.Config) (Store, error) {
	switch cfg.Blob.Backend {
	case config.BlobMinIO:
		return newMinIO(cfg)
	case config.BlobFS:
		return newFS(cfg)
	default:
		return nil, fmt.Errorf("unknown blob backend %q", cfg.Blob.Backend)
	}
}
