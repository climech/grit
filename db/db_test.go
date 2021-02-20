package db

import (
	"io/ioutil"
	"os"
	"testing"
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
		t.Fatalf("couldn't create date node successor: %v", err)
	}
	if succId != 3 {
		t.Fatalf("got successor ID = %d, want 3", succId)
	}

	// ID 2 should be our date node.
	if _, err := d.CreateEdge(1, 2); err == nil {
		t.Fatalf("created edge with date node as dest; err = nil, want non-nil")
	}
}

func TestAutodeleteDateNode(t *testing.T) {
	d := setupDB(t)
	defer tearDB(t, d)

	datestr := "2020-01-01"

	// Delete last successor.
	{
		succID, err := d.CreateSuccessorOfDateNode(datestr, "test")
		if err != nil {
			t.Fatalf("couldn't create date node successor: %v", err)
		}
		dateNode, err := d.GetNodeByName(datestr)
		if err != nil {
			t.Fatalf(`couldn't get node by name "%s": %v`, datestr, err)
		}
		if _, err := d.DeleteNode(succID); err != nil {
			t.Fatalf(`couldn't delete successor (%d): %v`, succID, err)
		}
		if n, err := d.GetNode(dateNode.ID); err != nil {
			t.Fatalf(`error getting node (%d): %v`, dateNode.ID, err)
		} else if n != nil {
			t.Error("date node still exists after deleting its only successor")
		}
	}

	// Unlink last successor.
	{
		succID, err := d.CreateSuccessorOfDateNode(datestr, "test")
		if err != nil {
			t.Fatalf("couldn't create date node successor: %v", err)
		}
		dateNode, err := d.GetNodeByName(datestr)
		if err != nil {
			t.Fatalf(`couldn't get node by name "%s": %v`, datestr, err)
		}
		if err := d.DeleteEdgeByEndpoints(dateNode.ID, succID); err != nil {
			t.Fatalf(`couldn't delete edge (%d) -> (%d): %v`,
				dateNode.ID, succID, err)
		}
		if n, err := d.GetNode(dateNode.ID); err != nil {
			t.Fatalf(`error getting node (%d): %v`, dateNode.ID, err)
		} else if n != nil {
			t.Error("date node still exists after unlinking its only successor")
		}
	}
}
