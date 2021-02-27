package multitree

type TaskStatus int

const (
	TaskStatusCompleted TaskStatus = iota
	TaskStatusInProgress
	TaskStatusInactive
)

func (s TaskStatus) String() string {
	switch s {
	case TaskStatusCompleted:
		return "completed"
	case TaskStatusInProgress:
		return "in progress"
	case TaskStatusInactive:
		return "inactive"
	default:
		panic("invalid task status")
	}
}

func (n *Node) Status() TaskStatus {
	if n.IsCompleted() {
		return TaskStatusCompleted
	} else if n.IsInProgress() {
		return TaskStatusInProgress
	}
	return TaskStatusInactive
}
