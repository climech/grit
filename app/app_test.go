package app

import (
	"testing"
	"os"
	"io/ioutil"
	"github.com/climech/grit/db"
)

// setupApp creates a new App and hooks it up to a test database.
func setupApp(t *testing.T) *App {
	tmpfile, err := ioutil.TempFile("", "grit_test_db")
	if err != nil {
		t.Fatalf("couldn't create temp file: %v", err)
	}
	tmpfile.Close() // We only want the name.
	d, err := db.New(tmpfile.Name())
	if err != nil {
		t.Fatalf("couldn't create db: %v", err)
	}
	return &App{Database: d}
}

func tearApp(t *testing.T, a *App) {
	a.Close()
	if err := os.Remove(a.Database.Filename); err != nil {
		t.Fatalf("error removing file: %v", err)
	}
}

// TestLoop fails if it's able to create a loop.
func TestLoop(t *testing.T) {
	a := setupApp(t)
	defer tearApp(t, a)

	node, err := a.AddRoot("test")
	if err != nil {
		t.Fatal("couldn't create node (1)")
	}

	// To create a loop, link node to itself.
	if _, err := a.LinkNodes(node.Id, node.Id); err == nil {
		t.Fatal("loop created (got: nil, want: *AppError)")
	} else if ae, ok := err.(*AppError); !ok {
		t.Fatalf("got error, but not of type *AppError: %v", err)
	} else if ok && ae.Code != ErrForbidden {
		t.Fatalf("got AppError, but ae.Code != ErrForbidden: %v", err)
	}
}

// TestBackEdge fails if it's able to create a back edge.
func TestBackEdge(t *testing.T) {
	a := setupApp(t)
	defer tearApp(t, a)

	// Create the graph:
	//
	//   [ ] test (1)
	//    └──[ ] test (2)
	//        └──[ ] test (3)
	//
	node1, err := a.AddRoot("test")
	if err != nil {
		t.Fatal("couldn't create node (1)")
	}
	node2, err := a.AddSuccessor("test", node1.Id)
	if err != nil {
		t.Fatal("couldn't create node (2)")
	}
	node3, err := a.AddSuccessor("test", node2.Id)
	if err != nil {
		t.Fatal("couldn't create node (3)")
	}

	// To make a cycle, link (3) to (1).
	if _, err := a.LinkNodes(node3.Id, node1.Id); err == nil {
		t.Fatal("a back edge was created")
	}
}

// TestForwardEdge fails if it's able to create a forward edge.
func TestForwardEdge(t *testing.T) {
	a := setupApp(t)
	defer tearApp(t, a)

	// Create the graph:
	//
	//   [ ] test (1)
	//    └──[ ] test (2)
	//        └──[ ] test (3)
	//
	node1, err := a.AddRoot("test")
	if err != nil {
		t.Fatal("couldn't create node (1)")
	}
	node2, err := a.AddSuccessor("test", node1.Id)
	if err != nil {
		t.Fatal("couldn't create node (2)")
	}
	node3, err := a.AddSuccessor("test", node2.Id)
	if err != nil {
		t.Fatal("couldn't create node (3)")
	}

	// To make a forward edge, link (1) to (3).
	if _, err := a.LinkNodes(node1.Id, node3.Id); err == nil {
		t.Fatal("forward edge successfully created")
	}
}

// TestCrossEdge fails if it cannot create a cross edge.
func TestCrossEdge(t *testing.T) {
	a := setupApp(t)
	defer tearApp(t, a)

	// Create the nodes:
	//
	//   [ ] test (1)
	//    └──[ ] test (2)
	//
	//   [ ] test (3)
	//
	root1, err := a.AddRoot("test")
	if err != nil {
		t.Fatal("couldn't create node (1)")
	}
	succ, err := a.AddSuccessor("test", root1.Id)
	if err != nil {
		t.Fatal("couldn't create node (2)")
	}
	root2, err := a.AddRoot("test")
	if err != nil {
		t.Fatal("couldn't create node (3)")
	}

	// To make a cross edge, link (3) to (2).
	if _, err := a.LinkNodes(root2.Id, succ.Id); err != nil {
		t.Fatalf("couldn't create a cross edge: %v", err)
	}
}

func TestStatusChange(t *testing.T) {
	a := setupApp(t)
	defer tearApp(t, a)

	// Create the graph:
	//
	//   [ ] test (1)
	//    ├──[ ] test (2)
	//    └──[ ] test (3)
	//        └──[ ] test (4)
	//
	node1, err := a.AddRoot("test")
	if err != nil {
		t.Fatal("couldn't create node (1)")
	}
	node2, err := a.AddSuccessor("test", node1.Id)
	if err != nil {
		t.Fatal("couldn't create node (2)")
	}
	node3, err := a.AddSuccessor("test", node1.Id)
	if err != nil {
		t.Fatal("couldn't create node (3)")
	}
	node4, err := a.AddSuccessor("test", node3.Id)
	if err != nil {
		t.Fatal("couldn't create node (4)")
	}

	// Checking (2) and (3) should cause (1) and (4) to be checked as well.
	//
	//   [x] test (1)
	//    ├──[x] test (2)
	//    └──[x] test (3)
	//        └──[x] test (4)
	//
	if err := a.CheckNode(node2.Id); err != nil {
		t.Fatalf("couldn't check successor (2): %v", err)
	}
	if err := a.CheckNode(node3.Id); err != nil {
		t.Fatalf("couldn't check successor (3): %v", err)
	}
	if root, err := a.GetGraph(node1.Id); err != nil {
		t.Fatalf("couldn't get graph: %v", err)
	} else {
		if n := root.Get(node2.Id); !n.Checked {
			t.Fatalf("unchecking node had no effect: %s", n)
		}
		if n := root.Get(node3.Id); !n.Checked {
			t.Fatalf("unchecking node had no effect: %s", n)
		}
		if !root.Checked {
			t.Fatal("checked all successors of node, but node still unchecked")
		}
		if !root.Get(node4.Id).Checked {
			t.Fatal("node checked, but successor is unchecked")
		}
	}

	// Unchecking (3) should cause (1) and (4) to be unchecked as well; (2) should
	// be left unchanged.
	//
	//   [ ] test (1)
	//    ├──[x] test (2)
	//    └──[ ] test (3)
	//        └──[ ] test (4)
	//
	if err := a.UncheckNode(node3.Id); err != nil {
		t.Fatalf("couldn't uncheck successor (3): %v", err)
	}
	if root, err := a.GetGraph(node1.Id); err != nil {
		t.Fatalf("couldn't get graph: %v", err)
	} else {
		if n := root.Get(node3.Id); n.Checked {
			t.Fatalf("unchecking node had no effect: %s", n)
		}
		if root.Checked {
			t.Fatal("node unchecked, but predecessor is checked")
		}
		if n := root.Get(node2.Id); !n.Checked {
			t.Fatalf("unchecking node changed node's sibling(!): %s", n)
		}
		if n := root.Get(node4.Id); n.Checked {
			t.Fatalf("node unchecked, but successor is checked: %s", n)
		}
	}
}
