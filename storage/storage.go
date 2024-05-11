package sql

import (
	trmgorm "github.com/avito-tech/go-transaction-manager/gorm"
	"github.com/slipneff/minor-bot/config"
	"github.com/slipneff/minor-bot/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Storage struct {
	db     *gorm.DB
	getter *trmgorm.CtxGetter
}

func New(db *gorm.DB, getter *trmgorm.CtxGetter) *Storage {
	return &Storage{
		db:     db,
		getter: getter,
	}
}

func NewSQLiteDB(cfg *config.Config) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open("dev.db"), &gorm.Config{
		TranslateError: true,
	})
}

func MustNewSQLite(cfg *config.Config) *gorm.DB {
	db, err := NewSQLiteDB(cfg)
	if err != nil {
		panic(err)
	}

	db.AutoMigrate(&models.User{}, &models.Respondent{}, &models.Customer{}, &models.Interview{})

	return db
}
