package database

import (
	"log"
	"os"
	"time"

	"github.com/zhunismp/tianfu-bank/services/transaction-service/core/infrastructure/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewPostgresDatabase(dbCfg config.DatabaseConfigProvider) *gorm.DB {
	dsn := dbCfg.GetDBDSN()

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Info,
			Colorful:      true,
		},
	)

	gormConfig := &gorm.Config{
		Logger:                                   gormLogger,
		DisableForeignKeyConstraintWhenMigrating: false,
	}

	gormDB, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		log.Fatalf("FATAL: Failed to connect to database: %v", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatalf("FATAL: Failed to get underlying sql.DB: %v", err)
	}

	if err = sqlDB.Ping(); err != nil {
		log.Fatalf("FATAL: Failed to ping database: %v", err)
	}

	log.Println("INFO: Database connection established successfully.")
	return gormDB
}

func ShutdownDatabase(gormDB *gorm.DB) {
	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Printf("ERROR: Failed to get sql.DB for shutdown: %v", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		log.Printf("ERROR: Failed to close database: %v", err)
		return
	}
	log.Println("INFO: Database connection closed successfully.")
}
