package dal

import (
	"database/sql"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"quai-transfer/config"
)

var (
	InterDB *gorm.DB
)

func DBInit(config *config.Config) {
	var (
		err   error
		sqlDB *sql.DB
	)

	type DbItem struct {
		DSN string
		DB  **gorm.DB
	}
	dbConfigs := []DbItem{
		{config.InterDSN, &InterDB},
	}

	for _, dbItem := range dbConfigs {
		if dbItem.DSN != "" {
			log.Println(dbItem.DSN)
			if *dbItem.DB, err = gorm.Open(postgres.Open(dbItem.DSN), &gorm.Config{}); err != nil {
				log.Fatal(err)
			}

			if sqlDB, err = (*dbItem.DB).DB(); err != nil {
				log.Fatal(err)
			}

			// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
			sqlDB.SetMaxIdleConns(10)

			// SetMaxOpenConns sets the maximum number of open connections to the database.
			sqlDB.SetMaxOpenConns(80)

			// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
			sqlDB.SetConnMaxLifetime(5 * time.Minute)
		}
	}

}
