package entity

import "time"

type Command struct {
	Name    string
	Command string
	Role    *string
}

type LogEntry struct {
	Stream string
	Data   string
}

type CommandExecution struct {
	ExecId    int
	CommandId int
	ExecTime  time.Time
	ExitCode  *int
	Log       []LogEntry
}

type ExecutionHistoryEntry struct {
	ExecId      int
	Time        time.Time
	CommandName string
}
