package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Endpoint, AccessKey, SecretKey, HotBucket, ColdBucket string
	UseSSL                                               bool
	MaxHotAge                                            time.Duration
}

type TieredStore struct {
	client    *minio.Client
	hot       string
	cold      string
	maxHotAge time.Duration
}

func New(cfg Config) (*TieredStore, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		return &TieredStore{hot: cfg.HotBucket, cold: cfg.ColdBucket, maxHotAge: defaultHotAge(cfg.MaxHotAge)}, nil
	}
	endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")
	client, err := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""), Secure: cfg.UseSSL})
	if err != nil {
		return nil, err
	}
	return &TieredStore{client: client, hot: cfg.HotBucket, cold: cfg.ColdBucket, maxHotAge: defaultHotAge(cfg.MaxHotAge)}, nil
}

func defaultHotAge(value time.Duration) time.Duration {
	if value <= 0 { return 30 * 24 * time.Hour }
	return value
}

func (s *TieredStore) Enabled() bool { return s != nil && s.client != nil && s.hot != "" }
func (s *TieredStore) bucket(cold bool) string {
	if cold && s.cold != "" { return s.cold }
	return s.hot
}

func (s *TieredStore) EnsureBuckets(ctx context.Context) error {
	if !s.Enabled() { return nil }
	for _, bucket := range []string{s.hot, s.cold} {
		if bucket == "" { continue }
		exists, err := s.client.BucketExists(ctx, bucket)
		if err != nil { return err }
		if !exists {
			if err := s.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil { return err }
		}
	}
	return nil
}

func (s *TieredStore) Upload(ctx context.Context, object string, reader io.Reader, size int64, contentType string, cold bool) (minio.UploadInfo, error) {
	if !s.Enabled() { return minio.UploadInfo{}, fmt.Errorf("tiered storage is not configured") }
	tier := "hot"; if cold { tier = "cold" }
	return s.client.PutObject(ctx, s.bucket(cold), object, reader, size, minio.PutObjectOptions{ContentType: contentType, UserMetadata: map[string]string{"x-amz-meta-tier": tier}})
}

func (s *TieredStore) PresignedGet(ctx context.Context, object string, cold bool, expiry time.Duration) (*url.URL, error) {
	if !s.Enabled() { return nil, fmt.Errorf("tiered storage is not configured") }
	if expiry <= 0 { expiry = 15 * time.Minute }
	return s.client.PresignedGetObject(ctx, s.bucket(cold), object, expiry, nil)
}

func (s *TieredStore) Download(ctx context.Context, object string, cold bool) (*minio.Object, error) {
	if !s.Enabled() { return nil, fmt.Errorf("tiered storage is not configured") }
	return s.client.GetObject(ctx, s.bucket(cold), object, minio.GetObjectOptions{})
}

func (s *TieredStore) MigrateOldObjects(ctx context.Context) error {
	if !s.Enabled() || s.cold == "" { return nil }
	objects := s.client.ListObjects(ctx, s.hot, minio.ListObjectsOptions{Recursive: true})
	for item := range objects {
		if item.Err != nil { return item.Err }
		if time.Since(item.LastModified) < s.maxHotAge { continue }
		_, err := s.client.CopyObject(ctx, minio.CopyDestOptions{Bucket: s.cold, Object: item.Key}, minio.CopySrcOptions{Bucket: s.hot, Object: item.Key})
		if err != nil { return err }
		if err := s.client.RemoveObject(ctx, s.hot, item.Key, minio.RemoveObjectOptions{}); err != nil { return err }
	}
	return nil
}
