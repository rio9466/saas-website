package primary

import "time"

// pageViewModel maps the page_views table.
type pageViewModel struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	Path      string    `gorm:"column:path"`
	Source    string    `gorm:"column:source"`
	Locale    string    `gorm:"column:locale"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (pageViewModel) TableName() string { return "page_views" }
