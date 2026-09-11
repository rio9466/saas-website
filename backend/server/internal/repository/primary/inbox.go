package primary

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/contact"
	"github.com/rio9466/easy-admin/server/internal/domain/media"
	"gorm.io/gorm"
)

// InboxRepository persists contact submissions and media metadata in primary
// PostgreSQL.
type InboxRepository struct {
	db *gorm.DB
}

// NewInboxRepository constructs an InboxRepository backed by db.
func NewInboxRepository(db *gorm.DB) *InboxRepository {
	return &InboxRepository{db: db}
}

func (r *InboxRepository) session(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func contactFromModel(m contactSubmissionModel) contact.Submission {
	return contact.Submission{
		ID:        m.ID,
		Name:      m.Name,
		Email:     m.Email,
		Company:   m.Company,
		Message:   m.Message,
		Locale:    m.Locale,
		Status:    m.Status,
		SourceIP:  m.SourceIP,
		UserAgent: m.UserAgent,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func mediaFromModel(m mediaAssetModel) media.Asset {
	return media.Asset{
		ID:           m.ID,
		URL:          m.URL,
		OriginalName: m.OriginalName,
		MIME:         m.MIME,
		SizeBytes:    m.SizeBytes,
		Width:        m.Width,
		Height:       m.Height,
		CreatedBy:    m.CreatedBy,
		CreatedAt:    m.CreatedAt,
	}
}

// --- contact submissions ---------------------------------------------------

// CreateContactSubmission inserts one submission and populates its id and
// timestamps.
func (r *InboxRepository) CreateContactSubmission(ctx context.Context, in *contact.Submission) error {
	if in == nil {
		return fmt.Errorf("contact submission is nil")
	}
	now := time.Now().UTC()
	model := contactSubmissionModel{
		Name:      in.Name,
		Email:     in.Email,
		Company:   in.Company,
		Message:   in.Message,
		Locale:    in.Locale,
		Status:    in.Status,
		SourceIP:  in.SourceIP,
		UserAgent: in.UserAgent,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := r.session(ctx).Create(&model).Error; err != nil {
		return wrapDBError(err)
	}
	in.ID = model.ID
	in.CreatedAt = now
	in.UpdatedAt = now
	return nil
}

// GetContactSubmission loads one submission by id.
func (r *InboxRepository) GetContactSubmission(ctx context.Context, id int64) (*contact.Submission, error) {
	var model contactSubmissionModel
	if err := r.session(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, wrapDBError(err)
	}
	submission := contactFromModel(model)
	return &submission, nil
}

// ListContactSubmissions returns submissions ordered newest first, optionally
// filtered by status. pageSize <= 0 returns every matching row.
func (r *InboxRepository) ListContactSubmissions(ctx context.Context, status string, page, pageSize int) ([]contact.Submission, int64, error) {
	db := r.session(ctx).Model(&contactSubmissionModel{})
	if s := strings.TrimSpace(status); s != "" {
		db = db.Where("status = ?", s)
	}
	models, total, err := listOrdered[contactSubmissionModel](db, "created_at DESC, id DESC", page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]contact.Submission, 0, len(models))
	for _, m := range models {
		out = append(out, contactFromModel(m))
	}
	return out, total, nil
}

// UpdateContactSubmissionStatus changes only the status column.
func (r *InboxRepository) UpdateContactSubmissionStatus(ctx context.Context, id int64, status string) error {
	res := r.session(ctx).Model(&contactSubmissionModel{}).Where("id = ?", id).
		Updates(map[string]any{"status": status, "updated_at": time.Now().UTC()})
	if res.Error != nil {
		return wrapDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// --- media assets ----------------------------------------------------------

// CreateMediaAsset inserts one media row and populates its id and timestamp.
func (r *InboxRepository) CreateMediaAsset(ctx context.Context, asset *media.Asset) error {
	if asset == nil {
		return fmt.Errorf("media asset is nil")
	}
	now := time.Now().UTC()
	model := mediaAssetModel{
		URL:          asset.URL,
		OriginalName: asset.OriginalName,
		MIME:         asset.MIME,
		SizeBytes:    asset.SizeBytes,
		Width:        asset.Width,
		Height:       asset.Height,
		CreatedBy:    asset.CreatedBy,
		CreatedAt:    now,
	}
	if err := r.session(ctx).Create(&model).Error; err != nil {
		return wrapDBError(err)
	}
	asset.ID = model.ID
	asset.CreatedAt = now
	return nil
}

// GetMediaAsset loads one media row by id.
func (r *InboxRepository) GetMediaAsset(ctx context.Context, id int64) (*media.Asset, error) {
	var model mediaAssetModel
	if err := r.session(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, wrapDBError(err)
	}
	asset := mediaFromModel(model)
	return &asset, nil
}

// ListMediaAssets returns media rows ordered newest first.
func (r *InboxRepository) ListMediaAssets(ctx context.Context, page, pageSize int) ([]media.Asset, int64, error) {
	models, total, err := listOrdered[mediaAssetModel](r.session(ctx).Model(&mediaAssetModel{}), "created_at DESC, id DESC", page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]media.Asset, 0, len(models))
	for _, m := range models {
		out = append(out, mediaFromModel(m))
	}
	return out, total, nil
}

// DeleteMediaAsset removes one media row by id.
func (r *InboxRepository) DeleteMediaAsset(ctx context.Context, id int64) error {
	res := r.session(ctx).Where("id = ?", id).Delete(&mediaAssetModel{})
	if res.Error != nil {
		return wrapDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
