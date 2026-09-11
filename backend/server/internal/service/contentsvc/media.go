package contentsvc

import (
	"bytes"
	"context"
	"io"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	"github.com/rio9466/easy-admin/server/internal/domain/content"
	"github.com/rio9466/easy-admin/server/internal/domain/media"
)

// mediaURLPrefix is the public path under which stored media is served.
const mediaURLPrefix = "/media/"

// UploadMedia validates one multipart upload, stores its bytes, and records the
// media row under a pending-first audit event. A failed record write removes
// the just-stored object so no orphan is left behind.
func (s *Service) UploadMedia(ctx context.Context, actor Actor, originalName string, declaredSize int64, body io.Reader) (*media.Asset, error) {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return nil, err
	}
	if declaredSize > media.MaxUploadBytes {
		return nil, validationError("file exceeds the 5MB limit")
	}
	data, err := io.ReadAll(io.LimitReader(body, media.MaxUploadBytes+1))
	if err != nil {
		return nil, mapError(err)
	}
	if int64(len(data)) > media.MaxUploadBytes {
		return nil, validationError("file exceeds the 5MB limit")
	}
	if len(data) == 0 {
		return nil, validationError("file is required")
	}
	mimeType, ok := media.DetectMIME(data)
	if !ok {
		return nil, validationError("unsupported file type")
	}
	width, height := media.Dimensions(mimeType, data)

	storedName := uuid.NewString() + media.Extension(mimeType)
	if err := s.mediaStore.Put(ctx, storedName, bytes.NewReader(data)); err != nil {
		return nil, mapError(err)
	}

	asset := &media.Asset{
		URL:          mediaURLPrefix + storedName,
		OriginalName: sanitizeOriginalName(originalName),
		MIME:         mimeType,
		SizeBytes:    int64(len(data)),
		Width:        width,
		Height:       height,
		CreatedBy:    actor.AdminID,
	}
	details := map[string]any{
		"mime":       mimeType,
		"size_bytes": int64(len(data)),
		"width":      width,
		"height":     height,
	}
	id, err := s.createAudited(ctx, actor, ActionMediaUpload, ResourceMedia, details, func() (int64, error) {
		if createErr := s.inbox.CreateMediaAsset(ctx, asset); createErr != nil {
			return 0, createErr
		}
		return asset.ID, nil
	})
	if err != nil {
		if deleteErr := s.mediaStore.Delete(storedName); deleteErr != nil {
			s.logger.Error("failed to remove orphaned media object",
				"stored_name", storedName, "request_id", actor.RequestID, "error", deleteErr)
		}
		return nil, err
	}
	asset.ID = id
	return asset, nil
}

// ListMedia returns a paginated media page, newest first.
func (s *Service) ListMedia(ctx context.Context, actor Actor, page, pageSize int) (adminauth.Page[media.Asset], error) {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return adminauth.Page[media.Asset]{}, err
	}
	items, total, err := s.inbox.ListMediaAssets(ctx, page, pageSize)
	if err != nil {
		return adminauth.Page[media.Asset]{}, mapError(err)
	}
	return adminauth.Page[media.Asset]{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// DeleteMedia removes one media row and its stored bytes under a pending-first
// audit event.
func (s *Service) DeleteMedia(ctx context.Context, actor Actor, id int64) error {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return err
	}
	asset, err := s.inbox.GetMediaAsset(ctx, id)
	if err != nil {
		return mapError(err)
	}
	if err := s.withPendingAudit(ctx, actor, ActionMediaDelete, ResourceMedia, idString(id), nil, func() error {
		return s.inbox.DeleteMediaAsset(ctx, id)
	}); err != nil {
		return err
	}
	if name, ok := mediaFilename(asset.URL); ok {
		if deleteErr := s.mediaStore.Delete(name); deleteErr != nil {
			s.logger.Error("failed to remove media object after row deletion",
				"stored_name", name, "media_id", idString(id), "request_id", actor.RequestID, "error", deleteErr)
		}
	}
	return nil
}

// OpenMedia resolves one stored media object for public serving. Only a single
// path segment is accepted, so traversal is impossible.
func (s *Service) OpenMedia(filename string) (media.StoredFile, error) {
	name := strings.TrimSpace(filename)
	if name == "" || name != path.Base(name) || name == "." || name == ".." {
		return nil, apperr.New(404, apperr.CodeNotFound, "not found", apperr.ErrNotFound)
	}
	file, err := s.mediaStore.Open(name)
	if err != nil {
		return nil, apperr.New(404, apperr.CodeNotFound, "not found", apperr.ErrNotFound)
	}
	return file, nil
}

func mediaFilename(url string) (string, bool) {
	if !strings.HasPrefix(url, mediaURLPrefix) {
		return "", false
	}
	name := strings.TrimPrefix(url, mediaURLPrefix)
	if name == "" || name != path.Base(name) {
		return "", false
	}
	return name, true
}

// sanitizeOriginalName keeps only the basename and caps its length so client
// input never reaches storage or HTML unescaped.
func sanitizeOriginalName(raw string) string {
	name := path.Base(strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/")))
	if name == "." || name == "/" {
		name = ""
	}
	if len(name) > 255 {
		name = name[:255]
	}
	return name
}
