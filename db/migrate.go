package db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func migrateFrom0(db *sql.DB) error {
	createNodes := `
		CREATE TABLE nodes (
			node_id INTEGER PRIMARY KEY,
			node_name VARCHAR(100) NOT NULL,
			node_alias VARCHAR(100) DEFAULT NULL,
			node_created INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			node_completed INTEGER DEFAULT NULL,

			UNIQUE(node_alias)
		)`

	createLinks := `
		CREATE TABLE links (
			link_id INTEGER PRIMARY KEY,
			origin_id INTEGER NOT NULL,
			dest_id INTEGER NOT NULL,

			FOREIGN KEY (origin_id)
				REFERENCES nodes (node_id)
				ON DELETE CASCADE

			FOREIGN KEY (dest_id)
				REFERENCES nodes (node_id)
				ON DELETE CASCADE

			CHECK(origin_id != dest_id)
			UNIQUE(origin_id, dest_id)
		)`

	if _, err := db.Exec(createNodes); err != nil {
		return err
	}
	if _, err := db.Exec(createLinks); err != nil {
		return err
	}

	return nil
}

// migrationFuncs is a slice of functions that incrementally migrate the DB from
// one version to the next. The length of this slice determines the latest known
// database version. The first "migration" initializes an empty DB.
var migrationFuncs = []func(*sql.DB) error{
	migrateFrom0,
}

// migrate checks if the underlying database is up-to-date, and migrates
// the data if needed. It returns an error if there's an IO problem or
// Grit doesn't recognize the DB version.
func (d *Database) migrate() error {
	v, err := d.getUserVersion()
	if err != nil {
		return err
	}
	if v < 0 {
		return fmt.Errorf("Corrupted database (negative user_version).")
	}

	current := int64(len(migrationFuncs))
	if v > current {
		return fmt.Errorf("Database version is not supported by this version of " +
			"Grit -- try upgrading to the latest release.")
	}
	for v < current {
		if err := migrationFuncs[v](d.DB); err != nil {
			return err
		}
		v++
		if err := d.setUserVersion(v); err != nil {
			return err
		}
	}

	return nil
}
