package app

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

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

func ptrValueToString(ptr interface{}) string {
	v := reflect.ValueOf(ptr)
	if v.IsNil() {
		return "nil"
	}
	return fmt.Sprintf("%v\n", reflect.Indirect(v))
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
	if _, err := a.LinkNodes(node.ID, node.ID); err == nil {
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
	node2, err := a.AddSuccessor("test", node1.ID)
	if err != nil {
		t.Fatal("couldn't create node (2)")
	}
	node3, err := a.AddSuccessor("test", node2.ID)
	if err != nil {
		t.Fatal("couldn't create node (3)")
	}

	// To make a cycle, link (3) to (1).
	if _, err := a.LinkNodes(node3.ID, node1.ID); err == nil {
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
	node2, err := a.AddSuccessor("test", node1.ID)
	if err != nil {
		t.Fatal("couldn't create node (2)")
	}
	node3, err := a.AddSuccessor("test", node2.ID)
	if err != nil {
		t.Fatal("couldn't create node (3)")
	}

	// To make a forward edge, link (1) to (3).
	if _, err := a.LinkNodes(node1.ID, node3.ID); err == nil {
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
	succ, err := a.AddSuccessor("test", root1.ID)
	if err != nil {
		t.Fatal("couldn't create node (2)")
	}
	root2, err := a.AddRoot("test")
	if err != nil {
		t.Fatal("couldn't create node (3)")
	}

	// To make a cross edge, link (3) to (2).
	if _, err := a.LinkNodes(root2.ID, succ.ID); err != nil {
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
	node2, err := a.AddSuccessor("test", node1.ID)
	if err != nil {
		t.Fatal("couldn't create node (2)")
	}
	node3, err := a.AddSuccessor("test", node1.ID)
	if err != nil {
		t.Fatal("couldn't create node (3)")
	}
	node4, err := a.AddSuccessor("test", node3.ID)
	if err != nil {
		t.Fatal("couldn't create node (4)")
	}

	// Checking (3) and (2) should cause (1) and (4) to be checked as well.
	//
	//   [x] test (1)
	//    ├──[x] test (2)
	//    └──[x] test (3)
	//        └──[x] test (4)
	//
	if err := a.CheckNode(node3.ID); err != nil {
		t.Fatalf("couldn't check successor (3): %v", err)
	}
	time.Sleep(1 * time.Second) // to make the timestamps different
	if err := a.CheckNode(node2.ID); err != nil {
		t.Fatalf("couldn't check successor (2): %v", err)
	}

	if root, err := a.GetGraph(node1.ID); err != nil {
		t.Fatalf("couldn't get graph: %v", err)
	} else {
		if n := root.Get(node2.ID); !n.IsCompleted() {
			t.Errorf("checking node had no effect: %s", n)
		}
		if n := root.Get(node3.ID); !n.IsCompleted() {
			t.Errorf("checking node had no effect: %s", n)
		}
		if !root.IsCompleted() {
			t.Error("checked all successors of root, but root.IsCompleted() = false")
		}
		if !root.Get(node4.ID).IsCompleted() {
			t.Error("node checked, but successor is still unchecked")
		}

		c1 := root.Completed
		c2 := root.Get(node2.ID).Completed
		c3 := root.Get(node3.ID).Completed
		c4 := root.Get(node4.ID).Completed

		if !reflect.DeepEqual(c1, c2) {
			t.Errorf("backpropped completion time should be the same as in "+
				"last checked successor; want %v, got %v",
				ptrValueToString(c2), ptrValueToString(c1))
		}
		if !reflect.DeepEqual(c3, c4) {
			t.Errorf("successor should inherit the completion time from its checked "+
				"predecessor; want %v, got %v", ptrValueToString(c3), ptrValueToString(c4))
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
	if err := a.UncheckNode(node3.ID); err != nil {
		t.Fatalf("couldn't uncheck successor (3): %v", err)
	}
	if root, err := a.GetGraph(node1.ID); err != nil {
		t.Fatalf("couldn't get graph: %v", err)
	} else {
		if n := root.Get(node3.ID); n.IsCompleted() {
			t.Fatalf("unchecking node had no effect: %s", n)
		}
		if root.IsCompleted() {
			t.Fatal("node unchecked, but predecessor is checked")
		}
		if n := root.Get(node2.ID); !n.IsCompleted() {
			t.Fatalf("unchecking node changed node's sibling(!): %s", n)
		}
		if n := root.Get(node4.ID); n.IsCompleted() {
			t.Fatalf("node unchecked, but successor is checked: %s", n)
		}
	}
}
