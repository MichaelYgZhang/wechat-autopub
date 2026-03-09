package domain

import "time"

type LogLevel string

const (
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

type PublishLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ArticleID uint      `json:"article_id" gorm:"index;not null"`
	Level     LogLevel  `json:"level" gorm:"size:10;not null"`
	Stage     string    `json:"stage" gorm:"size:50;not null"`
	Message   string    `json:"message" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
}

type PublishLogRepository interface {
	Create(log *PublishLog) error
	ListByArticleID(articleID uint) ([]PublishLog, error)
}
