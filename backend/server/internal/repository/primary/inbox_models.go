package primary

import "time"

// contactSubmissionModel maps the contact_submissions table.
type contactSubmissionModel struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	Name      string    `gorm:"column:name"`
	Email     string    `gorm:"column:email"`
	Company   string    `gorm:"column:company"`
	Message   string    `gorm:"column:message"`
	Locale    string    `gorm:"column:locale"`
	Status    string    `gorm:"column:status"`
	SourceIP  string    `gorm:"column:source_ip"`
	UserAgent string    `gorm:"column:user_agent"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (contactSubmissionModel) TableName() string { return "contact_submissions" }

// mediaAssetModel maps the media_assets table.
type mediaAssetModel struct {
	ID           int64     `gorm:"column:id;primaryKey"`
	URL          string    `gorm:"column:url"`
	OriginalName string    `gorm:"column:original_name"`
	MIME         string    `gorm:"column:mime"`
	SizeBytes    int64     `gorm:"column:size_bytes"`
	Width        int       `gorm:"column:width"`
	Height       int       `gorm:"column:height"`
	CreatedBy    int64     `gorm:"column:created_by"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (mediaAssetModel) TableName() string { return "media_assets" }
