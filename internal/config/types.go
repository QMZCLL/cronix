package config

import "github.com/QMZCLL/cronix/internal/task"

type Config struct {
	Tasks  []task.Task `json:"tasks"`
	LogDir string      `json:"log_dir"`
}
