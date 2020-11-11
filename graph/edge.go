package graph

import (
	"fmt"
	"github.com/fatih/color"
)

type Edge struct {
	Id int64
	OriginId int64
	DestId int64
}

func NewEdge(originId, destId int64) *Edge {
	return &Edge{OriginId: originId, DestId: destId}
}

func (e *Edge) String() string {
	accent := color.New(color.FgCyan).SprintFunc()
	id1 := accent(fmt.Sprintf("(%d)", e.OriginId))
	id2 := accent(fmt.Sprintf("(%d)", e.DestId))
	return fmt.Sprintf("%s -> %s", id1, id2)
}
