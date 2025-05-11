package database

import (
	"database/sql"
	"fmt"
)

func NewDBConn(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("can't establish connection to DB. DBConnPath:%s,err: %w", dbPath, err)
	}

	if err = db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("can't ping DB. DBConnPath: %s,err: %w", dbPath, err)
	}

	return db, nil
}

func GetTestingDBConn() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "test.db?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("can't establish connection to test DB. Err: %w", err)
	}

	if err = db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("can't ping DB. Err: %v", err)
	}

	return db, nil
}
