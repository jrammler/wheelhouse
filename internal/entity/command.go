package entity

import "time"

type Command struct {
	Name    string  `json:"name"`
	Command string  `json:"command"`
	Role    *string `json:"role,omitempty"`
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
