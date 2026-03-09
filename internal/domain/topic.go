package domain

import "time"

type Topic struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:100;not null"`
	Prompt    string    `json:"prompt" gorm:"type:text;not null"`
	Author    string    `json:"author" gorm:"size:100"`
	Active    bool      `json:"active" gorm:"default:true;index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TopicRepository interface {
	Create(topic *Topic) error
	GetByID(id uint) (*Topic, error)
	Update(topic *Topic) error
	Delete(id uint) error
	List(offset, limit int) ([]Topic, int64, error)
	GetRandomActive() (*Topic, error)
}
