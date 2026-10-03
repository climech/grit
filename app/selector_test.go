package app

import (
	"errors"
	"testing"
)

func errCode(err error) (ErrCode, bool) {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae.Code, true
	}
	return 0, false
}

func TestParseSelector(t *testing.T) {
	valid := []struct {
		in   string
		want selectorKind
	}{
		{"1", selectID},
		{"2020-01-01", selectDate},
		{"myalias", selectAlias},
		{"0", selectAlias}, // not a positive ID, so it falls through to alias
	}
	for _, tc := range valid {
		sel, err := ParseSelector(tc.in)
		if err != nil {
			t.Errorf("ParseSelector(%q): unexpected error: %v", tc.in, err)
			continue
		}
		if sel.kind != tc.want {
			t.Errorf("ParseSelector(%q): got kind %d, want %d", tc.in, sel.kind, tc.want)
		}
	}

	if _, err := ParseSelector(""); err == nil {
		t.Error("ParseSelector(\"\"): want error")
	} else if code, _ := errCode(err); code != ErrInvalidSelector {
		t.Errorf("ParseSelector(\"\"): got code %v, want ErrInvalidSelector", code)
	}
}

func TestInvalidSelectors(t *testing.T) {
	a := setupApp(t)
	defer tearApp(t, a)

	for name, sel := range map[string]Selector{"zero value": {}, "non-positive ID": ByID(0)} {
		_, err := a.GetGraph(sel)
		if code, ok := errCode(err); !ok || code != ErrInvalidSelector {
			t.Errorf("%s: got %v, want ErrInvalidSelector", name, err)
		}
	}
}

func TestUnknownAlias(t *testing.T) {
	a := setupApp(t)
	defer tearApp(t, a)

	sel, _ := ParseSelector("nope")
	if _, err := a.AddChild("x", sel); err == nil {
		t.Fatal("AddChild under unknown alias succeeded")
	} else if code, _ := errCode(err); code != ErrNotFound {
		t.Errorf("got code %v, want ErrNotFound", code)
	}
}

func TestMissingDateNode(t *testing.T) {
	a := setupApp(t)
	defer tearApp(t, a)

	date, _ := ParseSelector("2020-01-01")

	// Reading a missing date yields a mock node and creates nothing.
	if n, err := a.GetGraph(date); err != nil || n == nil || n.Name != "2020-01-01" {
		t.Fatalf("GetGraph(missing date) = %v, %v; want mock d-node", n, err)
	}
	if roots, _ := a.GetDateNodes(); len(roots) != 0 {
		t.Fatalf("GetGraph created a date node")
	}

	// Mutating a missing date reports not found.
	for name, err := range map[string]error{
		"CheckNode":  a.CheckNode(date),
		"RemoveNode": func() error { _, e := a.RemoveNode(date); return e }(),
		"Rename":     a.RenameNode(date, "x"),
	} {
		if code, ok := errCode(err); !ok || code != ErrNotFound {
			t.Errorf("%s on missing date: got %v, want ErrNotFound", name, err)
		}
	}

	// Adding a child creates the date node implicitly.
	child, err := a.AddChild("task", date)
	if err != nil {
		t.Fatalf("AddChild under missing date: %v", err)
	}
	if ps := child.Parents(); len(ps) != 1 || ps[0].Name != "2020-01-01" {
		t.Errorf("child parents = %v, want the date node", ps)
	}
}

func TestLinkFromMissingDate(t *testing.T) {
	a := setupApp(t)
	defer tearApp(t, a)

	root, err := a.AddRoot("root")
	if err != nil {
		t.Fatal(err)
	}
	date, _ := ParseSelector("2020-01-01")
	if _, err := a.LinkNodes(date, ByID(root.ID)); err != nil {
		t.Fatalf("couldn't link from missing date: %v", err)
	}
	other, _ := ParseSelector("2021-01-01")
	if _, err := a.LinkNodes(ByID(root.ID), other); err == nil {
		t.Error("linking to a missing date node should fail")
	} else if code, _ := errCode(err); code != ErrNotFound {
		t.Errorf("got code %v, want ErrNotFound", code)
	}
}

func TestErrorMapping(t *testing.T) {
	a := setupApp(t)
	defer tearApp(t, a)

	if _, err := a.AddChild("x", ByID(99)); err == nil {
		t.Error("AddChild with missing parent succeeded")
	} else if code, _ := errCode(err); code != ErrNotFound {
		t.Errorf("missing parent: got code %v, want ErrNotFound", code)
	}

	n1, _ := a.AddRoot("a")
	n2, _ := a.AddRoot("b")
	if err := a.SetAlias(n1.ID, "dup"); err != nil {
		t.Fatal(err)
	}
	if err := a.SetAlias(n2.ID, "dup"); err == nil {
		t.Error("duplicate alias accepted")
	} else if code, _ := errCode(err); code != ErrForbidden {
		t.Errorf("duplicate alias: got code %v, want ErrForbidden", code)
	}

	if err := a.CheckNode(ByID(99)); err == nil {
		t.Error("CheckNode on missing ID succeeded")
	} else if code, _ := errCode(err); code != ErrNotFound {
		t.Errorf("CheckNode missing: got code %v, want ErrNotFound", code)
	}

	if _, err := a.AddRoot(""); err == nil {
		t.Error("AddRoot with empty name succeeded")
	} else if code, _ := errCode(err); code != ErrInvalidName {
		t.Errorf("AddRoot empty name: got code %v, want ErrInvalidName", code)
	}
}
