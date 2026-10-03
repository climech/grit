package db

import (
	"errors"
	"os"
	"testing"
)

func setupDB(t *testing.T) *Database {
	tmpfile, err := os.CreateTemp("", "grit_test_db")
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

	rootID, err := d.CreateNode("test root", 0)
	if err != nil {
		t.Fatalf("couldn't create root: %v", err)
	}
	if rootID != 1 {
		t.Fatalf("got root ID = %d, want 1", rootID)
	}

	childID, err := d.CreateChildOfDateNode("2020-01-01", "test child")
	if err != nil {
		t.Fatalf("couldn't create child of date node: %v", err)
	}
	if childID != 3 {
		t.Fatalf("got child ID = %d, want 3", childID)
	}

	// ID 2 should be our date node.
	if _, err := d.CreateLink(1, 2); err == nil {
		t.Fatalf("created link with date node as dest; err = nil, want non-nil")
	}
}

func TestAutodeleteDateNode(t *testing.T) {
	d := setupDB(t)
	defer tearDB(t, d)

	datestr := "2020-01-01"

	// Delete last child.
	{
		childID, err := d.CreateChildOfDateNode(datestr, "test")
		if err != nil {
			t.Fatalf("couldn't create child of date node: %v", err)
		}
		dateNode, err := d.GetNodeByName(datestr)
		if err != nil {
			t.Fatalf(`couldn't get node by name "%s": %v`, datestr, err)
		}
		if _, err := d.DeleteNode(childID); err != nil {
			t.Fatalf(`couldn't delete child (%d): %v`, childID, err)
		}
		if n, err := d.GetNode(dateNode.ID); err != nil {
			t.Fatalf(`error getting node (%d): %v`, dateNode.ID, err)
		} else if n != nil {
			t.Error("date node still exists after deleting its only child")
		}
	}

	// Unlink last child.
	{
		childID, err := d.CreateChildOfDateNode(datestr, "test")
		if err != nil {
			t.Fatalf("couldn't create date node child: %v", err)
		}
		dateNode, err := d.GetNodeByName(datestr)
		if err != nil {
			t.Fatalf(`couldn't get node by name "%s": %v`, datestr, err)
		}
		if err := d.DeleteLinkByEndpoints(dateNode.ID, childID); err != nil {
			t.Fatalf(`couldn't delete link (%d) -> (%d): %v`,
				dateNode.ID, childID, err)
		}
		if n, err := d.GetNode(dateNode.ID); err != nil {
			t.Fatalf(`error getting node (%d): %v`, dateNode.ID, err)
		} else if n != nil {
			t.Error("date node still exists after unlinking its only child")
		}
	}
}

func TestSentinelErrors(t *testing.T) {
	d := setupDB(t)
	defer tearDB(t, d)

	if _, err := d.CreateNode("a", 99); !errors.Is(err, ErrNodeNotFound) {
		t.Errorf("CreateNode with missing parent: got %v, want ErrNodeNotFound", err)
	}
	if err := d.CheckNode(99); !errors.Is(err, ErrNodeNotFound) {
		t.Errorf("CheckNode on missing node: got %v, want ErrNodeNotFound", err)
	}
	if err := d.RenameNode(99, "x"); !errors.Is(err, ErrNodeNotFound) {
		t.Errorf("RenameNode on missing node: got %v, want ErrNodeNotFound", err)
	}
	if _, err := d.DeleteNode(99); !errors.Is(err, ErrNodeNotFound) {
		t.Errorf("DeleteNode on missing node: got %v, want ErrNodeNotFound", err)
	}
	if err := d.SetAlias(99, "x"); !errors.Is(err, ErrNodeNotFound) {
		t.Errorf("SetAlias on missing node: got %v, want ErrNodeNotFound", err)
	}

	id1, _ := d.CreateNode("a", 0)
	id2, _ := d.CreateNode("b", 0)
	if err := d.DeleteLinkByEndpoints(id1, id2); !errors.Is(err, ErrLinkNotFound) {
		t.Errorf("DeleteLinkByEndpoints without link: got %v, want ErrLinkNotFound", err)
	}

	if err := d.SetAlias(id1, "dup"); err != nil {
		t.Fatalf("couldn't set alias: %v", err)
	}
	if err := d.SetAlias(id2, "dup"); !errors.Is(err, ErrAliasExists) {
		t.Errorf("duplicate alias: got %v, want ErrAliasExists", err)
	}
}
