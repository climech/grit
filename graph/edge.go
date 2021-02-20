package graph

import (
	"fmt"

	"github.com/fatih/color"
)

type Edge struct {
	ID       int64
	OriginID int64
	DestID   int64
}

func NewEdge(originID, destID int64) *Edge {
	return &Edge{OriginID: originID, DestID: destID}
}

func (e *Edge) String() string {
	accent := color.New(color.FgCyan).SprintFunc()
	id1 := accent(fmt.Sprintf("(%d)", e.OriginID))
	id2 := accent(fmt.Sprintf("(%d)", e.DestID))
	return fmt.Sprintf("%s -> %s", id1, id2)
}
