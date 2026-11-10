package orm

import (
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

type SqliteConfig struct {
	Path string
	MaxIdleConns int
	MaxOpenConns int
}

func NewSqlite(sqliteConf *SqliteConfig, logwriter logger.Writer) *gorm.DB {
	path:=sqliteConf.Path
	if path=="" {path="./data/rustdeskapi.db"}
	db, err := gorm.Open(sqlite.Open("file:"+path+"?_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger: logger.New(
			logwriter, // io writer
			logger.Config{
				SlowThreshold:             time.Second, // Slow SQL threshold
				LogLevel:                  logger.Warn, // Log level
				IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
				ParameterizedQueries:      true,        // Don't include params in the SQL log
				Colorful:                  true,
			},
		),
	})
	if err != nil {
		panic(fmt.Errorf("open SQLite database: %w", err))
	}
	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Errorf("obtain SQLite connection pool: %w", err))
	}
	// SQLite permits one writer; bound the number of competing connections.
	maxOpen := sqliteConf.MaxOpenConns
	if maxOpen <= 0 || maxOpen > 4 {
		maxOpen = 4
	}
	maxIdle := sqliteConf.MaxIdleConns
	if maxIdle < 0 {
		maxIdle = 0
	}
	if maxIdle > maxOpen {
		maxIdle = maxOpen
	}
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)

	return db
}
