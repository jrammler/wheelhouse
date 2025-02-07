package entity

import "time"

type Command struct {
	Name    string  `json:"name"`
	Command string  `json:"command"`
	Id      string  `json:"-"`
	Role    *string `json:"role,omitempty"`
}

type LogEntry struct {
	Stream string
	Data   string
}

type CommandExecution struct {
	ExecId    int
	CommandId string
	ExecTime  time.Time
	ExitCode  *int
	Log       []LogEntry
}

type ExecutionHistoryEntry struct {
	ExecId      int
	Time        time.Time
	CommandName string
}
