package blob

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/johnnycube/openbeehive-app/server/internal/config"
)

type minioStore struct {
	c      *minio.Client
	bucket string
	pub    string
}

func newMinIO(cfg *config.Config) (Store, error) {
	c, err := minio.New(cfg.Blob.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Blob.AccessKey, cfg.Blob.SecretKey, ""),
		Secure: cfg.Blob.UseSSL,
	})
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	exists, err := c.BucketExists(ctx, cfg.Blob.Bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := c.MakeBucket(ctx, cfg.Blob.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}
	return &minioStore{c: c, bucket: cfg.Blob.Bucket, pub: cfg.Blob.PublicURL}, nil
}

func (m *minioStore) Put(ctx context.Context, key, ct string, r io.Reader, size int64) error {
	_, err := m.c.PutObject(ctx, m.bucket, key, r, size, minio.PutObjectOptions{ContentType: ct})
	return err
}

func (m *minioStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return m.c.GetObject(ctx, m.bucket, key, minio.GetObjectOptions{})
}

func (m *minioStore) Delete(ctx context.Context, key string) error {
	return m.c.RemoveObject(ctx, m.bucket, key, minio.RemoveObjectOptions{})
}

func (m *minioStore) PresignedPut(ctx context.Context, key, ct string, ttl time.Duration) (string, error) {
	u, err := m.c.PresignedPutObject(ctx, m.bucket, key, ttl)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (m *minioStore) PublicURL(key string) string {
	return m.pub + "/" + url.PathEscape(key)
}
