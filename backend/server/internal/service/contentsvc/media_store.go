package contentsvc

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/media"
)

// LocalDiskMediaStore is the default MediaStore implementation: media bytes are
// written to a configured directory and served from it. The directory is
// resolved to an absolute path once at construction so later working-directory
// changes cannot redirect writes.
type LocalDiskMediaStore struct {
	root string
}

var _ media.Store = (*LocalDiskMediaStore)(nil)

// NewLocalDiskMediaStore creates the directory when missing and returns a store
// rooted at it. An empty root defaults to "media".
func NewLocalDiskMediaStore(root string) (*LocalDiskMediaStore, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		root = "media"
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}
	return &LocalDiskMediaStore{root: abs}, nil
}

// Root returns the absolute storage directory (used by tests).
func (s *LocalDiskMediaStore) Root() string { return s.root }

// Put writes filename under the store root, replacing any existing object with
// the same name.
func (s *LocalDiskMediaStore) Put(ctx context.Context, filename string, content io.Reader) error {
	if err := validateStoredFilename(filename); err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(s.root, filename), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, content); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

// Open returns the stored object for reading.
func (s *LocalDiskMediaStore) Open(filename string) (media.StoredFile, error) {
	if err := validateStoredFilename(filename); err != nil {
		return nil, err
	}
	file, err := os.Open(filepath.Join(s.root, filename))
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	return &diskFile{File: file, info: info}, nil
}

// Delete removes the stored object; a missing object is not an error.
func (s *LocalDiskMediaStore) Delete(filename string) error {
	if err := validateStoredFilename(filename); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(s.root, filename)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// diskFile adapts *os.File to media.StoredFile.
type diskFile struct {
	*os.File
	info os.FileInfo
}

func (f *diskFile) ModTime() time.Time { return f.info.ModTime() }
func (f *diskFile) Size() int64        { return f.info.Size() }

// validateStoredFilename rejects anything that is not a single path segment.
func validateStoredFilename(name string) error {
	if name == "" || name == "." || name == ".." || name != filepath.Base(name) ||
		strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("invalid media filename")
	}
	return nil
}
