// Package grit/db implements the basic CRUD operations used to interact with
// grit data. All operations are atomic.
package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	DB       *sql.DB
	Filename string
}

func New(filename string) (*Database, error) {
	d := &Database{}
	if err := d.Open(filename); err != nil {
		return nil, err
	}
	if err := d.init(); err != nil {
		return nil, err
	}
	d.Filename = filename
	return d, nil
}

func (d *Database) init() error {
	if _, err := d.DB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return err
	}
	return d.migrate()
}

func (d *Database) getUserVersion() (int64, error) {
	row := d.DB.QueryRow(`PRAGMA user_version`)
	var version int64
	if err := row.Scan(&version); err != nil {
		return 0, err
	}
	return version, nil
}

func (d *Database) setUserVersion(version int64) error {
	// Using fmt.Sprintf -- driver doesn't parametrize values for PRAGMAs.
	query := fmt.Sprintf("PRAGMA user_version = %d", version)
	_, err := d.DB.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) Open(fp string) error {
	sqlite3db, err := sql.Open("sqlite3", fp)
	if err != nil {
		return err
	}
	d.DB = sqlite3db
	return nil
}

func (d *Database) Close() error {
	return d.DB.Close()
}

func (d *Database) beginTx() (*sql.Tx, error) {
	ctx := context.TODO()
	return d.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
}

func (d *Database) execTxFunc(f func(*sql.Tx) error) error {
	tx, err := d.beginTx()
	if err != nil {
		return err
	}
	if err := f(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
