package services

import (
	"context"
	"fmt"
	"io"
	nethttp "net/http"
	"net/url"
	"path"
	"strings"

	"github.com/google/uuid"

	"myevent-back/internal/dto"
	"myevent-back/internal/storage"
)

const uploadFormOverheadBytes int64 = 1 << 20

var (
	allowedUploadFolders = map[string]struct{}{
		"events/covers":  {},
		"events/gifts":   {},
		"events/gallery": {},
	}
	allowedUploadMIMEs = map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/webp": ".webp",
		"image/gif":  ".gif",
	}
)

type UploadService struct {
	storage          storage.Provider
	publicURL        string
	maxFileSizeBytes int64
}

func NewUploadService(storage storage.Provider, maxFileSizeBytes int64) *UploadService {
	if maxFileSizeBytes <= 0 {
		maxFileSizeBytes = 10 << 20
	}

	return &UploadService{
		storage:          storage,
		publicURL:        strings.TrimRight(strings.TrimSpace(storage.PublicURL()), "/"),
		maxFileSizeBytes: maxFileSizeBytes,
	}
}

func (s *UploadService) MaxBodySizeBytes() int64 {
	return s.maxFileSizeBytes + uploadFormOverheadBytes
}

func (s *UploadService) Upload(ctx context.Context, folder, filename string, file io.Reader) (*dto.UploadResponse, error) {
	folder, err := normalizeUploadFolder(folder)
	if err != nil {
		return nil, err
	}

	content, err := io.ReadAll(io.LimitReader(file, s.maxFileSizeBytes+1))
	if err != nil {
		return nil, err
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("%w: file is required", ErrValidation)
	}
	if int64(len(content)) > s.maxFileSizeBytes {
		return nil, fmt.Errorf("%w: file exceeds %d bytes", ErrValidation, s.maxFileSizeBytes)
	}

	contentType := nethttp.DetectContentType(content)
	extension, ok := allowedUploadMIMEs[contentType]
	if !ok {
		return nil, fmt.Errorf("%w: unsupported file type", ErrValidation)
	}

	key := path.Join(folder, uuid.NewString()+extension)
	url, err := s.storage.PutObject(ctx, key, content, contentType)
	if err != nil {
		return nil, err
	}

	_ = filename

	return &dto.UploadResponse{
		URL: url,
		Key: key,
	}, nil
}

func (s *UploadService) Delete(ctx context.Context, key string) error {
	key, err := normalizeUploadKey(key)
	if err != nil {
		return err
	}

	return s.storage.DeleteObject(ctx, key)
}

func (s *UploadService) ManagedKeyFromURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || s.publicURL == "" {
		return "", false
	}

	publicURL, err := url.Parse(s.publicURL)
	if err != nil {
		return "", false
	}

	assetURL, err := url.Parse(raw)
	if err != nil {
		return "", false
	}

	if !strings.EqualFold(assetURL.Scheme, publicURL.Scheme) || !strings.EqualFold(assetURL.Host, publicURL.Host) {
		return "", false
	}

	publicPath := strings.Trim(publicURL.Path, "/")
	assetPath := strings.Trim(assetURL.Path, "/")
	if publicPath != "" {
		if assetPath == publicPath || !strings.HasPrefix(assetPath, publicPath+"/") {
			return "", false
		}

		assetPath = strings.TrimPrefix(assetPath, publicPath+"/")
	}

	key, err := normalizeUploadKey(assetPath)
	if err != nil {
		return "", false
	}

	return key, true
}

func normalizeUploadFolder(folder string) (string, error) {
	cleaned := strings.Trim(strings.TrimSpace(folder), "/")
	if _, ok := allowedUploadFolders[cleaned]; !ok {
		return "", fmt.Errorf("%w: folder is not allowed", ErrValidation)
	}
	return cleaned, nil
}

func normalizeUploadKey(key string) (string, error) {
	cleaned := strings.Trim(strings.TrimSpace(key), "/")
	if cleaned == "" {
		return "", fmt.Errorf("%w: key is required", ErrValidation)
	}

	if path.Clean(cleaned) != cleaned || strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("%w: invalid upload key", ErrValidation)
	}

	parts := strings.Split(cleaned, "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("%w: invalid upload key", ErrValidation)
	}

	folder := strings.Join(parts[:2], "/")
	if _, ok := allowedUploadFolders[folder]; !ok {
		return "", fmt.Errorf("%w: invalid upload key", ErrValidation)
	}

	return cleaned, nil
}
