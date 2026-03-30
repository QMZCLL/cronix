package logger

import "time"

type RunRecord struct {
	TaskName  string    `json:"task_name"`
	StartedAt time.Time `json:"started_at"`
	ExitCode  int       `json:"exit_code"`
	LogFile   string    `json:"log_file"`
}
