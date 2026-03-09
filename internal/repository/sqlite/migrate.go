package sqlite

import (
	"github.com/MichaelYgZhang/wechat-autopub/internal/domain"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDB(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(
		&domain.Article{},
		&domain.Topic{},
		&domain.PublishLog{},
	); err != nil {
		return nil, err
	}

	return db, nil
}
