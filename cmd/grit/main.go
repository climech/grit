package main

import (
	"fmt"
	"os"

	"github.com/climech/grit/app"
	cli "github.com/jawher/mow.cli"
)

func main() {
	c := cli.App(app.AppName, "A multitree-based personal task manager")
	c.Version("v version", fmt.Sprintf("%s %s", app.AppName, app.Version))

	c.Command("add", "Add a new node", cmdAdd)
	c.Command("alias", "Create alias", cmdAlias)
	c.Command("unalias", "Remove alias", cmdUnalias)
	c.Command("tree", "Print tree representation rooted at node", cmdTree)
	c.Command("check", "Mark node(s) as completed", cmdCheck)
	c.Command("uncheck", "Revert node status to inactive", cmdUncheck)
	c.Command("link", "Create a link from one node to another", cmdLink)
	c.Command("unlink", "Remove an existing link between two nodes", cmdUnlink)
	c.Command("list ls", "List children of selected node", cmdList)
	c.Command("list-dates lsd", "List all date nodes", cmdListDates)
	c.Command("rename", "Rename a node", cmdRename)
	c.Command("remove rm", "Remove node(s)", cmdRemove)
	c.Command("import", "Import nodes from tab-indented lines", cmdImport)
	c.Command("stat", "Display node information", cmdStat)

	args := os.Args
	if len(args) == 1 {
		// Run `tree` implicitly.
		args = append(os.Args, "tree")
	}

	c.Run(args)
}
