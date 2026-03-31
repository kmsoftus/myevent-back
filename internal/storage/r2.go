package storage

import (
	"bytes"
	"context"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Config struct {
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Region          string
	Endpoint        string
	PublicURL       string
}

type R2Storage struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

func NewR2Storage(ctx context.Context, cfg R2Config) (*R2Storage, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.UsePathStyle = true
		options.BaseEndpoint = aws.String(strings.TrimRight(cfg.Endpoint, "/"))
	})

	return &R2Storage{
		client:    client,
		bucket:    cfg.Bucket,
		publicURL: normalizeR2PublicURL(cfg.PublicURL, cfg.Endpoint, cfg.Bucket),
	}, nil
}

func (s *R2Storage) PutObject(ctx context.Context, key string, content []byte, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(content),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", err
	}

	return s.publicURL + "/" + strings.TrimLeft(key, "/"), nil
}

func (s *R2Storage) DeleteObject(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (s *R2Storage) PublicURL() string {
	return s.publicURL
}

func normalizeR2PublicURL(publicURL, endpoint, bucket string) string {
	publicURL = strings.TrimRight(strings.TrimSpace(publicURL), "/")
	if publicURL == "" {
		return ""
	}

	publicParsed, err := url.Parse(publicURL)
	if err != nil {
		return publicURL
	}

	if trimmedPath := strings.Trim(publicParsed.Path, "/"); trimmedPath != "" {
		return publicURL
	}

	endpointParsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil {
		return publicURL
	}

	bucket = strings.Trim(strings.TrimSpace(bucket), "/")
	endpointPath := strings.Trim(endpointParsed.Path, "/")
	if bucket == "" || endpointPath != bucket {
		return publicURL
	}

	publicParsed.Path = "/" + bucket
	return strings.TrimRight(publicParsed.String(), "/")
}
