package domain

import "time"

type ArticleStatus string

const (
	ArticleStatusPending    ArticleStatus = "pending"
	ArticleStatusPublishing ArticleStatus = "publishing"
	ArticleStatusPublished  ArticleStatus = "published"
	ArticleStatusFailed     ArticleStatus = "failed"
)

type Article struct {
	ID            uint          `json:"id" gorm:"primaryKey"`
	Title         string        `json:"title" gorm:"size:255;not null"`
	Author        string        `json:"author" gorm:"size:100"`
	Content       string        `json:"content" gorm:"type:text"`
	Digest        string        `json:"digest" gorm:"size:500"`
	ThumbMediaID  string        `json:"thumb_media_id" gorm:"size:255"`
	TopicID       *uint         `json:"topic_id" gorm:"index"`
	Status        ArticleStatus `json:"status" gorm:"size:20;default:pending;index"`
	DraftMediaID  string        `json:"draft_media_id" gorm:"size:255"`
	PublishID     string        `json:"publish_id" gorm:"size:255"`
	ArticleURL    string        `json:"article_url" gorm:"size:512"`
	ErrorMessage  string        `json:"error_message" gorm:"type:text"`
	ScheduledAt   *time.Time    `json:"scheduled_at"`
	PublishedAt   *time.Time    `json:"published_at"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

type ArticleRepository interface {
	Create(article *Article) error
	GetByID(id uint) (*Article, error)
	Update(article *Article) error
	Delete(id uint) error
	List(offset, limit int) ([]Article, int64, error)
	ListByStatus(status ArticleStatus, offset, limit int) ([]Article, int64, error)
	GetPendingForPublish() (*Article, error)
	CountByStatus() (map[ArticleStatus]int64, error)
}
