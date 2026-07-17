package blob

import (
	"context"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/johnnycube/openbeehive-app/server/internal/config"
)

// fsStore stores blobs flat in the filesystem. For presigned uploads there is
// no external service - instead the app server itself provides an
// upload endpoint (see HTTP handler); the URL carries a short-lived token.
type fsStore struct {
	base string
	pub  string
}

func newFS(cfg *config.Config) (Store, error) {
	if err := os.MkdirAll(cfg.Blob.BaseDir, 0o755); err != nil {
		return nil, err
	}
	return &fsStore{base: cfg.Blob.BaseDir, pub: cfg.Blob.PublicURL}, nil
}

func (f *fsStore) path(key string) string {
	// key may contain "/" -> subfolders
	return filepath.Join(f.base, filepath.FromSlash(key))
}

func (f *fsStore) Put(ctx context.Context, key, ct string, r io.Reader, size int64) error {
	p := f.path(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	dst, err := os.Create(p)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, r)
	return err
}

func (f *fsStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return os.Open(f.path(key))
}

func (f *fsStore) Delete(ctx context.Context, key string) error {
	return os.Remove(f.path(key))
}

// PresignedPut: in FS mode the URL points to our own upload endpoint.
// A real signature token (HMAC + expiry) is created in the HTTP layer.
func (f *fsStore) PresignedPut(ctx context.Context, key, ct string, ttl time.Duration) (string, error) {
	return f.pub + "/upload?key=" + url.QueryEscape(key), nil
}

func (f *fsStore) PublicURL(key string) string {
	return f.pub + "/" + url.PathEscape(key)
}
