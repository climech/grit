package multitree

import (
	"fmt"

	"github.com/fatih/color"
)

type Link struct {
	ID       int64
	OriginID int64
	DestID   int64
}

func NewLink(originID, destID int64) *Link {
	return &Link{OriginID: originID, DestID: destID}
}

func (l *Link) String() string {
	accent := color.New(color.FgCyan).SprintFunc()
	id1 := accent(fmt.Sprintf("(%d)", l.OriginID))
	id2 := accent(fmt.Sprintf("(%d)", l.DestID))
	return fmt.Sprintf("%s -> %s", id1, id2)
}
