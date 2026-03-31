package storage

import "context"

type Provider interface {
	PutObject(ctx context.Context, key string, content []byte, contentType string) (string, error)
	DeleteObject(ctx context.Context, key string) error
	PublicURL() string
}
