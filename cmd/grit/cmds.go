package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/climech/grit/app"
	"github.com/climech/grit/multitree"

	"github.com/fatih/color"
	cli "github.com/jawher/mow.cli"
)

func cmdAdd(cmd *cli.Cmd) {
	cmd.Spec = "[ -p=<predecessor> | -r ] NAME_PARTS..."
	today := time.Now().Format("2006-01-02")

	var (
		nameParts = cmd.StringsArg("NAME_PARTS", nil,
			"strings to be joined together to form the node's name")
		predecessor = cmd.StringOpt("p predecessor", today,
			"predecessor to attach the node to")
		makeRoot = cmd.BoolOpt("r root", false,
			"create a root node")
	)

	cmd.Action = func() {
		a, err := app.New()
		if err != nil {
			die(err)
		}
		defer a.Close()

		name := strings.Join(*nameParts, " ")

		if *makeRoot {
			node, err := a.AddRoot(name)
			if err != nil {
				dief("Couldn't create node: %v\n", err)
			}
			color.Cyan("(%d)", node.ID)
		} else {
			node, err := a.AddChild(name, *predecessor)
			if err != nil {
				dief("Couldn't create node: %v\n", err)
			}
			parents := node.Parents()
			accent := color.New(color.FgCyan).SprintFunc()
			if parents[0].Name == today {
				accent = color.New(color.FgYellow).SprintFunc()
			}
			highlighted := accent(fmt.Sprintf("(%d)", node.ID))
			fmt.Printf("(%d) -> %s\n", parents[0].ID, highlighted)
		}
	}
}

func cmdTree(cmd *cli.Cmd) {
	cmd.Spec = "[-i] [-C] [NODE]"
	var (
		idsort    = cmd.BoolOpt("i id-sort", false, "sort by id instead of name")
		nochecked = cmd.BoolOpt("C no-checked", false, "filter out checked nodes")
		selector  = cmd.StringArg("NODE", "", "node selector")
	)
	cmd.Action = func() {
		a, err := app.New()
		if err != nil {
			die(err)
		}
		defer a.Close()
		var nodes []*multitree.Node

		if *selector == "" {
			// List all dates by default.
			dnodes, err := a.GetDateNodes()
			if err != nil {
				die(err)
			}
			for _, d := range dnodes {
				// Get as part of multitree for accurate status.
				n, err := a.GetGraph(d.ID)
				if err != nil {
					die(capitalize(err.Error()))
				}
				if n == nil {
					continue
				}
				nodes = append(nodes, n)
			}
		} else {
			n, err := a.GetGraph(*selector)
			if err != nil {
				die(capitalize(err.Error()))
			}
			if n == nil {
				die("Node does not exist")
			}
			nodes = append(nodes, n)
		}

		for _, node := range nodes {
			if *nochecked && !node.IsCompleted() {
				fmt.Print(node.Name)
			}
			node.TraverseDescendants(func(current *multitree.Node, _ func()) {
				if *idsort {
					multitree.SortNodesByID(current.Children())
				} else {
					multitree.SortNodesByName(current.Children())
				}
			})
			fmt.Print(node.StringTree(*nochecked))
		}
	}
}

func cmdList(cmd *cli.Cmd) {
	cmd.Spec = "[-i] [-C] [NODE]"
	var (
		idsort    = cmd.BoolOpt("i id-sort", false, "sort by id instead of name")
		nochecked = cmd.BoolOpt("C no-checked", false, "filter out checked nodes")
		selector  = cmd.StringArg("NODE", "", "node selector")
	)
	cmd.Action = func() {
		a, err := app.New()
		if err != nil {
			die(err)
		}
		defer a.Close()
		var nodes []*multitree.Node

		if *selector == "" {
			// List roots by default.
			roots, err := a.GetRoots()
			if err != nil {
				die(err)
			}
			for _, r := range roots {
				// Get as part of multitree for accurate status.
				n, err := a.GetGraph(r.ID)
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
			nodes = node.Children()
		}

		if *idsort {
			multitree.SortNodesByID(nodes)
		} else {
			multitree.SortNodesByName(nodes)
		}
		for _, n := range nodes {
			if !*nochecked || !n.IsCompleted() {
				fmt.Println(n)
			}
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
	cmd.Spec = "ORIGIN TARGETS..."
	var (
		origin  = cmd.StringArg("ORIGIN", "", "origin selector")
		targets = cmd.StringsArg("TARGETS", nil, "target selector(s)")
	)
	cmd.Action = func() {
		a, err := app.New()
		if err != nil {
			die(err)
		}
		defer a.Close()

		for _, t := range *targets {
			if _, err := a.LinkNodes(*origin, t); err != nil {
				errf("Couldn't create link (%s) -> (%s): %v\n", *origin, t, err)
			}
		}
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
			n, err := a.GetGraph(d.ID)
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
	cmd.Spec = "NODE NAME_PARTS..."
	var (
		selector  = cmd.StringArg("NODE", "", "node selector")
		nameParts = cmd.StringsArg("NAME_PARTS", nil,
			"strings forming the new node name")
	)
	cmd.Action = func() {
		a, err := app.New()
		if err != nil {
			die(err)
		}
		defer a.Close()
		name := strings.Join(*nameParts, " ")
		if err := a.RenameNode(*selector, name); err != nil {
			dief("Couldn't rename node: %v", err)
		}
	}
}

func cmdAlias(cmd *cli.Cmd) {
	cmd.Spec = "NODE_ID ALIAS"
	var (
		selector = cmd.StringArg("NODE_ID", "", "node ID selector")
		alias    = cmd.StringArg("ALIAS", "", "alias string")
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
		recursive = cmd.BoolOpt("r recursive", false,
			"remove node and all its descendants")
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

		appendErr := func(sel string, err error) {
			errs = append(errs, fmt.Errorf("Couldn't remove %s: %v", sel, err))
		}

		for _, sel := range *selectors {
			if *recursive {
				removed, err := a.RemoveNodeRecursive(sel)
				if err != nil {
					appendErr(sel, err)
					continue
				}
				for _, node := range removed {
					msgs = append(msgs, fmt.Sprintf("Removed: %v ", node))
				}
			} else {
				removed, err := a.GetGraph(sel)
				if err != nil {
					appendErr(sel, err)
					continue
				}
				orphaned, err := a.RemoveNode(sel)
				if err != nil {
					appendErr(sel, err)
					continue
				}
				msgs = append(msgs, fmt.Sprintf("Removed: %v ", removed))
				for _, node := range orphaned {
					msgs = append(msgs, fmt.Sprintf("Orphaned: %v ", node))
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
		filename = cmd.StringArg("FILENAME", "",
			"file containing tab-indented lines")
		predecessor = cmd.StringOpt("p predecessor", today,
			"predecessor for the tree root(s)")
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

		roots, err := multitree.ImportTrees(reader)
		if err != nil {
			dief("Import error: %v", err)
		}

		var errs []error
		var treesTotal, nodesTotal int

		for _, root := range roots {
			var id int64
			var err error
			if *makeRoot {
				id, err = a.AddRootTree(root)
			} else {
				id, err = a.AddChildTree(root, *predecessor)
			}
			if err != nil {
				e := fmt.Errorf("Couldn't import tree: %v", err)
				errs = append(errs, e)
				continue
			}

			if g, err := a.GetGraph(id); err != nil {
				errs = append(errs, err)
			} else {
				fmt.Print(g.StringTree(false))
				treesTotal++
				nodesTotal += len(g.Tree().All())
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

		parents := node.Parents()
		children := node.Children()

		if len(parents)+len(children) > 0 {
			fmt.Println(node.StringNeighbors())
		}

		status := node.Status().String()
		leaves := node.Leaves()
		done := 0
		total := len(leaves)
		for _, leaf := range leaves {
			if leaf.IsCompleted() {
				done++
			}
		}
		if total > 0 {
			status += fmt.Sprintf(" (%d/%d)", done, total)
		}

		// Make name bold if root.
		name := node.Name
		if len(parents) == 0 {
			bold := color.New(color.Bold).SprintFunc()
			name = bold(name)
		}

		fmt.Printf("ID: %d\n", node.ID)
		fmt.Printf("Name: %s\n", name)
		fmt.Printf("Status: %s\n", status)
		fmt.Printf("Parents: %d\n", len(parents))
		fmt.Printf("Children: %d\n", len(children))

		if node.Alias != "" {
			fmt.Printf("Alias: %s\n", node.Alias)
		}

		timeFmt := "2006-01-02 15:04:05"
		fmt.Printf("Created: %s\n", time.Unix(node.Created, 0).Format(timeFmt))
		if node.IsCompleted() {
			fmt.Printf("Checked: %s\n", time.Unix(*node.Completed, 0).Format(timeFmt))
		}

	}
}
