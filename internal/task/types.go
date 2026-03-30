package task

import "time"

type Task struct {
	Name        string            `json:"name"`
	Command     string            `json:"command"`
	CronExpr    string            `json:"cron_expr"`
	Enabled     bool              `json:"enabled"`
	RunOnce     bool              `json:"run_once"`
	Envs        map[string]string `json:"envs"`
	CreatedAt   time.Time         `json:"created_at"`
	LastRunAt   *time.Time        `json:"last_run_at"`
	Description string            `json:"description"`
}
