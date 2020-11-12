package db

import (
	"testing"
	"os"
	"io/ioutil"
)

func setupDB(t *testing.T) *Database {
	tmpfile, err := ioutil.TempFile("", "grit_test_db")
	if err != nil {
		t.Fatalf("couldn't create temp file: %v", err)
	}
	tmpfile.Close() // We only want the name.
	d, err := New(tmpfile.Name())
	if err != nil {
		t.Fatalf("couldn't create db: %v", err)
	}
	return d
}

func tearDB(t *testing.T, d *Database) {
	d.Close()
	if err := os.Remove(d.Filename); err != nil {
		t.Fatalf("error removing file: %v", err)
	}
}

func TestLinkToDateNode(t *testing.T) {
	d := setupDB(t)
	defer tearDB(t, d)

	rootId, err := d.CreateNode("test root", 0)
	if err != nil {
		t.Fatalf("couldn't create root: %v", err)
	}
	if rootId != 1 {
		t.Fatalf("got root ID = %d, want 1", rootId)
	}

	succId, err := d.CreateSuccessorOfDateNode("2020-01-01", "test successor")
	if err != nil {
		t.Fatalf("couldn't create root: %v", err)
	}
	if succId != 3 {
		t.Fatalf("got successor ID = %d, want 3", succId)
	}

	// ID 2 should be our date node.
	if _, err := d.CreateEdge(1, 2); err == nil {
		t.Fatalf("created edge with date node as dest; err = nil, want non-nil")
	}
}