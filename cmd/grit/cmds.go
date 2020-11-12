package main

import (
	"os"
	"fmt"
	"time"
	"io"
	"strconv"

	"github.com/climech/grit/app"
	"github.com/climech/grit/graph"

	"github.com/jawher/mow.cli"
	"github.com/fatih/color"
)

func cmdAdd(cmd *cli.Cmd) {
	cmd.Spec = "[ -p=<predecessor> | -r ] NAME"
	today := time.Now().Format("2006-01-02")
	var (
		name = cmd.StringArg("NAME", "", "node name")
		predecessor = cmd.StringOpt(
			"p predecessor",
			today,
			"predecessor to attach the node to",
		)
		makeRoot = cmd.BoolOpt("r root", false, "create a root node, not a successor")
	)
	cmd.Action = func() {
		a, err := app.New()
		if err != nil {
			die(err)
		}
		defer a.Close()

		if *makeRoot {
			node, err := a.AddRoot(*name)
			if err != nil {
				dief("Couldn't create node: %v\n", err)
			}
			msg := fmtCreate(fmt.Sprintf("Created root: %s", node))
			fmt.Println(msg)
		} else {
			node, err := a.AddSuccessor(*name, *predecessor)
			if err != nil {
				dief("Couldn't create node: %v\n", err)
			}
			accent := color.New(color.FgCyan).SprintFunc()
			if node.Predecessors[0].Name == today {
				accent = color.New(color.FgYellow).SprintFunc()
			}
			idstr := accent(fmt.Sprintf("(%d)", node.Predecessors[0].Id))
			msg := fmt.Sprintf("Created node: %s -> %s", idstr, node)
			fmt.Println(fmtCreate(msg))
		}
	}
}

func cmdTree(cmd *cli.Cmd) {
	cmd.Spec = "[NODE]"
	today := time.Now().Format("2006-01-02")
	var (
		selector = cmd.StringArg("NODE", today, "node selector")
	)
	cmd.Action = func() {
		a, err := app.New()
		if err != nil {
			die(err)
		}
		defer a.Close()

		node, err := a.GetGraph(*selector)
		if err != nil {
			die(capitalize(err.Error()))
		}
		if node == nil {
			die("Node does not exist")
		}
		fmt.Print(node.TreeString())
	}
}

func cmdList(cmd *cli.Cmd) {
	cmd.Spec = "[NODE]"
	var (
		selector = cmd.StringArg("NODE", "", "node selector")
	)
	cmd.Action = func() {
		a, err := app.New()
		if err != nil {
			die(err)
		}
		defer a.Close()
		var nodes []*graph.Node

		if *selector == "" {
			roots, err := a.GetRoots()
			if err != nil {
				die(err)
			}
			for _, r := range roots {
				// Get as part of graph for accurate status.
				n, err := a.GetGraph(r.Id)
				if err != nil {
					die(err)
				}
				if n == nil {
					continue
				}
				nodes = append(nodes, n)
			}
		} else {
			node, err := a.GetGraph(*selector)
			if err != nil {
				die(capitalize(err.Error()))
			}
			if node == nil {
				die("Node does not exist")
			}
			nodes = node.Successors
		}

		for _, n := range nodes {
			fmt.Println(n)
		}
	}
}

func cmdCheck(cmd *cli.Cmd) {
	cmd.Spec = "NODE..."
	var (
		selectors = cmd.StringsArg("NODE", nil, "node selector(s)")
	)
	cmd.Action = func() {
		a, err := app.New()
		if err != nil {
			die(err)
		}
		defer a.Close()
		for _, sel := range *selectors {
			if err := a.CheckNode(sel); err != nil {
				dief("Couldn't check node: %v", err)
			}
		}
	}
}

func cmdUncheck(cmd *cli.Cmd) {
	cmd.Spec = "NODE..."
	var (
		selectors = cmd.StringsArg("NODE", nil, "node selector(s)")
	)
	cmd.Action = func() {
		a, err := app.New()
		if err != nil {
			die(err)
		}
		defer a.Close()
		for _, sel := range *selectors {
			if err := a.UncheckNode(sel); err != nil {
				dief("Couldn't uncheck node: %v", err)
			}
		}
	}
}

func cmdLink(cmd *cli.Cmd) {
	cmd.Spec = "ORIGIN TARGET"
	var (
		origin = cmd.StringArg("ORIGIN", "", "origin selector")
		target = cmd.StringArg("TARGET", "", "target selector")
	)
	cmd.Action = func() {
		a, err := app.New()
		if err != nil {
			die(err)
		}
		defer a.Close()

		edge, err := a.LinkNodes(*origin, *target)
		if err != nil {
			dief("Couldn't link nodes: %v\n", err)
		}
		msg := fmtCreate(fmt.Sprintf("Created edge: %s", edge))
		fmt.Println(msg)
	}
}

func cmdUnlink(cmd *cli.Cmd) {
	cmd.Spec = "ORIGIN TARGET"
	var (
		origin = cmd.StringArg("ORIGIN", "", "origin selector")
		target = cmd.StringArg("TARGET", "", "target selector")
	)
	cmd.Action = func() {
		a, err := app.New()
		if err != nil {
			die(err)
		}
		defer a.Close()

		if err := a.UnlinkNodes(*origin, *target); err != nil {
			dief("Couldn't unlink nodes: %v\n", err)
		}
		accent := color.New(color.FgCyan).SprintFunc()
		sel1 := accent(fmt.Sprintf("(%s)", *origin))
		sel2 := accent(fmt.Sprintf("(%s)", *target))
		msg := fmtDelete(fmt.Sprintf("Deleted edge: %s -> %s", sel1, sel2))
		fmt.Println(msg)
	}
}

func cmdListRoots(cmd *cli.Cmd) {
	cmd.Action = func() {
		a, err := app.New()
		if err != nil {
			die(err)
		}
		defer a.Close()

		roots, err := a.GetRoots()
		if err != nil {
			die(err)
		}
		for _, r := range roots {
			// Get the nodes as members of their graphs to get accurate status.
			n, err := a.GetGraph(r.Id)
			if err != nil {
				die(err)
			}
			if n == nil {
				continue
			}
			fmt.Println(n)
		}
	}
}

func cmdListDates(cmd *cli.Cmd) {
	cmd.Action = func() {
		a, err := app.New()
		if err != nil {
			die(err)
		}
		defer a.Close()

		dnodes, err := a.GetDateNodes()
		if err != nil {
			die(err)
		}
		for _, d := range dnodes {
			// Get the nodes as members of their graphs to get accurate status.
			n, err := a.GetGraph(d.Id)
			if err != nil {
				die(err)
			}
			if n == nil {
				continue
			}
			fmt.Println(n)
		}
	}
}

func cmdRename(cmd *cli.Cmd) {
	cmd.Spec = "NODE NAME"
	var (
		selector = cmd.StringArg("NODE", "", "node selector")
		name = cmd.StringArg("NAME", "", "new name for NODE")
	)
	cmd.Action = func() {
		a, err := app.New()
		if err != nil {
			die(err)
		}
		defer a.Close()
		if err := a.RenameNode(*selector, *name); err != nil {
			dief("Couldn't rename node: %v", err)
		}
	}
}

func cmdAlias(cmd *cli.Cmd) {
	cmd.Spec = "NODE_ID ALIAS"
	var (
		selector = cmd.StringArg("NODE_ID", "", "node ID selector")
		alias = cmd.StringArg("ALIAS", "", "alias string")
	)
	cmd.Action = func() {
		a, err := app.New()
		if err != nil {
			die(err)
		}
		defer a.Close()

		id, err := strconv.ParseInt(*selector, 10, 64)
		if err != nil {
			dief("Selector must be an integer")
		}
		if err := a.SetAlias(id, *alias); err != nil {
			dief("Couldn't set alias: %v", err)
		}
	}
}

func cmdUnalias(cmd *cli.Cmd) {
	cmd.Spec = "NODE_ID"
	var (
		selector = cmd.StringArg("NODE_ID", "", "node ID selector")
	)
	cmd.Action = func() {
		a, err := app.New()
		if err != nil {
			die(err)
		}
		defer a.Close()

		id, err := strconv.ParseInt(*selector, 10, 64)
		if err != nil {
			dief("Selector must be an integer")
		}
		if err := a.SetAlias(id, ""); err != nil {
			dief("Couldn't unset alias: %v", err)
		}
	}
}

func cmdRemove(cmd *cli.Cmd) {
	cmd.Spec = "[-r] [-v] NODE..."
	var (
		selectors = cmd.StringsArg("NODE", nil, "node selector(s)")
		recursive = cmd.BoolOpt("r recursive", false, "remove node and all its descendants")
		verbose = cmd.BoolOpt("v verbose", false, "print each removed node")
	)
	cmd.Action = func() {
		a, err := app.New()
		if err != nil {
			die(err)
		}
		defer a.Close()

		var msgs []string
		var errs []error

		for _, sel := range *selectors {
			if *recursive {
				deleted, err := a.RemoveNodeRecursive(sel)
				if err != nil {
					errs = append(errs, fmt.Errorf("Couldn't remove %v: %v", sel, err))
					continue
				}
				for _, d := range deleted {
					msg := fmtDelete(fmt.Sprintf("Deleted node: %v ", d))
					msgs = append(msgs, msg)
				}
			} else {
				orphaned, err := a.RemoveNode(sel)
				if err != nil {
					errs = append(errs, fmt.Errorf("Couldn't remove %v: %v", sel, err))
				}
				for _, o := range orphaned {
					msg := fmtUpdate(fmt.Sprintf("Orphaned node: %v ", o))
					msgs = append(msgs, msg)
				}
			}
		}

		if *verbose {
			for _, msg := range msgs {
				fmt.Println(msg)
			}
		}
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, e)
		}
	}
}

func cmdImport(cmd *cli.Cmd) {
	cmd.Spec = "[ -p=<predecessor> | -r ] [FILENAME]"
	today := time.Now().Format("2006-01-02")
	var (
		filename = cmd.StringArg("FILENAME", "", "file containing tab-indented lines")
		predecessor = cmd.StringOpt("p predecessor", today, "predecessor for the tree root(s)")
		makeRoot = cmd.BoolOpt("r root", false, "create top-level tree(s)")
	)
	cmd.Action = func() {
		a, err := app.New()
		if err != nil {
			die(err)
		}
		defer a.Close()

		var reader io.Reader
		if *filename == "" {
			reader = os.Stdin
		} else {
			f, err := os.Open(*filename)
			if err != nil {
				dief("%s\n", capitalize(err.Error()))
			}
			defer f.Close()
			reader = f
		}

		var errs []error
		var treesTotal, nodesTotal int

		roots := graph.ImportNodes(reader)
		for _, root := range roots {
			var id int64
			var err error
			if *makeRoot {
				id, err = a.AddTree(root, 0)
			} else {
				id, err = a.AddTree(root, *predecessor)
			}
			if err != nil {
				e := fmt.Errorf("Couldn't import tree: %v", err)
				errs = append(errs, e)
				continue
			}

			if g, err := a.GetGraph(id); err != nil {
				errs = append(errs, err)
			} else {
				fmt.Print(g.TreeString())
				treesTotal++
				nodesTotal += len(g.Tree().GetAll())
			}
		}

		for _, e := range errs {
			fmt.Fprintln(os.Stderr, e)
		}
		fmt.Printf("Imported %d trees (%d nodes)\n", treesTotal, nodesTotal)
	}
}

func cmdStat(cmd *cli.Cmd) {
	cmd.Spec = "NODE"
	var (
		selector = cmd.StringArg("NODE", "", "node selector")
	)
	cmd.Action = func() {
		a, err := app.New()
		if err != nil {
			die(err)
		}
		defer a.Close()

		node, err := a.GetGraph(*selector)
		if err != nil {
			die(err)
		} else if node == nil {
			die("Node does not exist")
		}

		if len(node.Predecessors) + len(node.Successors) > 0 {
			fmt.Printf("\n%s\n", node.EdgeString())
		}

		status := node.Status().String()
		var done, total int
		node.Tree().EachAfter(func(n *graph.Node, _ int) {
			// Count the leaves.
			if len(n.Successors) == 0 {
				total++
				if n.Checked {
					done++
				}
			}
		})
		if total > 0 {
			status += fmt.Sprintf(" (%d/%d)", done, total)
		}

		// Make name bold if root.
		name := node.Name
		if len(node.Predecessors) == 0 {
			bold := color.New(color.Bold).SprintFunc()
			name = bold(name)
		}

		fmt.Printf("ID: %d\n", node.Id)
		fmt.Printf("Name: %s\n", name)
		fmt.Printf("Status: %s\n", status)
		fmt.Printf("Predecessors: %d\n", len(node.Predecessors))
		fmt.Printf("Successors: %d\n", len(node.Successors))
		if node.Alias != "" {
			fmt.Printf("Alias: %s\n", node.Alias)
		}
	}
}
