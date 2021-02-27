package multitree

import "fmt"

// validateNewLink creates deep copies of the nodes' graphs, connects the copied
// nodes, and checks if the resulting graph is a valid multitree.
func validateNewLink(origin, dest *Node) error {
	if origin.ID == 0 || dest.ID == 0 {
		panic("link endpoints must have IDs")
	}
	if origin.HasChild(dest) != dest.HasParent(origin) {
		panic("parent/child out of sync")
	}
	if origin.HasChild(dest) {
		return fmt.Errorf("link already exists")
	}
	if ValidateDateNodeName(dest.Name) == nil {
		return fmt.Errorf("cannot unroot date node")
	}

	// The nodes cannot belong to the same multitree. This ensures that no cycles
	// or diamonds are introduced into the digraph.
	if nodesOverlap(origin.All(), dest.All()) {
		return fmt.Errorf("nodes belong to the same multitree")
	}

	/*
		parent := origin.Copy()
		child := dest.Copy()
		parent.children = append(parent.children, child)
		child.parents = append(child.parents, parent)

		if parent.hasBackEdge() {
			return fmt.Errorf("cycles are not allowed")
		}
		if parent.hasDiamond() {
			return fmt.Errorf("diamonds are not allowed")
		}
	*/

	return nil
}

// LinkNodes creates a directed link from origin to dest, provided that the
// resulting graph will be a valid multitree. It returns an error otherwise.
// The given nodes are modified only if no error occurs.
func LinkNodes(origin, dest *Node) error {
	if err := validateNewLink(origin, dest); err != nil {
		return err
	}
	origin.children = append(origin.children, dest)
	dest.parents = append(dest.parents, origin)
	return nil
}

// LinkNodes removes an existing directed link between origin and dest. It
// returns an error if the link doesn't exist.
func UnlinkNodes(origin, dest *Node) error {
	if origin.HasChild(dest) != dest.HasParent(origin) {
		panic("parent/child out of sync")
	}
	if !origin.HasChild(dest) {
		return fmt.Errorf("link does not exist")
	}
	origin.children, _ = removeNode(origin.children, dest)
	dest.parents, _ = removeNode(dest.parents, origin)
	return nil
}
