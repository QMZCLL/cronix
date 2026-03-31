package main

import (
	"path/filepath"

	"github.com/QMZCLL/cronix/internal/config"
	"github.com/QMZCLL/cronix/internal/task"
	"github.com/spf13/cobra"
)

const version = "0.3.1"

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "cronix",
		Short:         "cronix - a TUI cron job manager",
		Long:          "cronix manages cron jobs through an interactive TUI.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(
		newInitCmd(),
		newAddCmd(),
		newEnableCmd(),
		newDisableCmd(),
		newRemoveCmd(),
		newListCmd(),
		newRunCmd(),
		newLogsCmd(),
		newTUICmd(),
	)

	return cmd
}

func wrappersDir() string {
	return filepath.Join(config.ConfigDir(), "wrappers")
}

func normalizeTasks(tasks []task.Task) []task.Task {
	if tasks == nil {
		return []task.Task{}
	}
	return tasks
}
