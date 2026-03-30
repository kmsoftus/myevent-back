package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type LocalStorage struct {
	baseDir   string
	publicURL string
}

func NewLocalStorage(baseDir, publicURL string) (*LocalStorage, error) {
	baseDir = filepath.Clean(baseDir)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}

	return &LocalStorage{
		baseDir:   baseDir,
		publicURL: strings.TrimRight(publicURL, "/"),
	}, nil
}

func (s *LocalStorage) PutObject(_ context.Context, key string, content []byte, _ string) (string, error) {
	targetPath, err := s.resolvePath(key)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(targetPath, content, 0o644); err != nil {
		return "", err
	}

	return s.publicURL + "/" + strings.TrimLeft(key, "/"), nil
}

func (s *LocalStorage) DeleteObject(_ context.Context, key string) error {
	targetPath, err := s.resolvePath(key)
	if err != nil {
		return err
	}

	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (s *LocalStorage) resolvePath(key string) (string, error) {
	cleanedKey := filepath.Clean(filepath.FromSlash(strings.TrimLeft(key, "/")))
	targetPath := filepath.Clean(filepath.Join(s.baseDir, cleanedKey))

	relativePath, err := filepath.Rel(s.baseDir, targetPath)
	if err != nil {
		return "", err
	}
	if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid storage key")
	}

	return targetPath, nil
}
