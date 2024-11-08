package dal

import (
	"database/sql"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	LocalDB SqliteDB
	log     = logging.Logger("database")
)

type SqliteDB struct {
	db *gorm.DB
}

func NewLocalDB(dsn string) (*SqliteDB, error) {
	var (
		err   error
		sqlDB *sql.DB
	)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	if sqlDB, err = db.DB(); err != nil {
		log.Fatal(err)
	}

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(80)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	//db.AutoMigrate(&WalletItem{})
	return &SqliteDB{
		db: db,
	}, nil
}

func (l *SqliteDB) Close() error {
	db, err := l.db.DB()
	if err != nil {
		log.Warn("get db err: ", err)
		return err
	}

	err = db.Close()
	if err != nil {
		log.Warn("db close err:", err)
		return err
	}
	return nil
}
