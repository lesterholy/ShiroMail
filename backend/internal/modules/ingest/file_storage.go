package ingest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LocalFileStorage struct {
	rootDir string
}

func NewLocalFileStorage(rootDir string) (*LocalFileStorage, error) {
	if strings.TrimSpace(rootDir) == "" {
		return nil, fmt.Errorf("mail storage root is required")
	}
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return nil, err
	}
	return &LocalFileStorage{rootDir: rootDir}, nil
}

func (s *LocalFileStorage) StoreRaw(_ context.Context, mailboxAddress string, sourceMessageID string, raw []byte) (string, error) {
	key := s.key("raw", mailboxAddress, sourceMessageID, "message.eml")
	if err := s.writeFile(key, raw); err != nil {
		return "", err
	}
	return key, nil
}

func (s *LocalFileStorage) StoreAttachment(_ context.Context, mailboxAddress string, sourceMessageID string, attachment InboundAttachment, index int) (StoredAttachment, error) {
	fileName := attachment.FileName
	if strings.TrimSpace(fileName) == "" {
		fileName = fmt.Sprintf("attachment-%d.bin", index+1)
	}

	key := s.key("attachments", mailboxAddress, sourceMessageID, fmt.Sprintf("%02d-%s", index+1, sanitizePathSegment(fileName)))
	if err := s.writeFile(key, attachment.Content); err != nil {
		return StoredAttachment{}, err
	}

	return StoredAttachment{
		FileName:    fileName,
		ContentType: attachment.ContentType,
		StorageKey:  key,
		SizeBytes:   int64(len(attachment.Content)),
	}, nil
}

func (s *LocalFileStorage) ReadFile(_ context.Context, key string) ([]byte, error) {
	return os.ReadFile(filepath.Join(s.rootDir, filepath.FromSlash(key)))
}

func (s *LocalFileStorage) DeleteFilesOlderThan(_ context.Context, before time.Time) error {
	if before.IsZero() {
		return nil
	}

	for _, topLevel := range []string{"raw", "attachments"} {
		root := filepath.Join(s.rootDir, topLevel)
		if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if info.IsDir() {
				return nil
			}
			if info.ModTime().After(before) {
				return nil
			}
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
			return nil
		}); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := removeEmptyDirectories(root); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (s *LocalFileStorage) key(kind string, mailboxAddress string, sourceMessageID string, fileName string) string {
	now := time.Now().UTC()
	parts := []string{
		kind,
		fmt.Sprintf("%04d", now.Year()),
		fmt.Sprintf("%02d", now.Month()),
		fmt.Sprintf("%02d", now.Day()),
		sanitizePathSegment(mailboxAddress),
		sanitizePathSegment(sourceMessageID),
		fileName,
	}
	return filepath.ToSlash(filepath.Join(parts...))
}

func (s *LocalFileStorage) writeFile(key string, content []byte) error {
	fullPath := filepath.Join(s.rootDir, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, content, 0o644)
}

func sanitizePathSegment(value string) string {
	replacer := strings.NewReplacer(
		"\\", "_",
		"/", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		"@", "_at_",
		" ", "_",
	)
	cleaned := strings.TrimSpace(replacer.Replace(value))
	if cleaned == "" {
		return "item"
	}
	return cleaned
}

func removeEmptyDirectories(root string) error {
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if err := removeEmptyDirectories(filepath.Join(root, entry.Name())); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	entries, err = os.ReadDir(root)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		if err := os.Remove(root); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
