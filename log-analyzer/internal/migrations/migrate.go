// Package migrations 数据库迁移
package migrations

import (
	"github.com/log-system/log-analyzer/internal/models"
	"gorm.io/gorm"
)

// Migrate 执行数据库迁移
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Rule{},
		&models.Condition{},
		&models.Action{},
		&models.RuleVersion{},
		&models.Strategy{},
		&models.StrategyRule{},
		&models.LogPattern{},
		&models.LogCluster{},
		&models.LogEntry{},
		&models.LogReport{},
	)
}
