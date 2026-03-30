package main

import (
	"fmt"

	"github.com/QMZCLL/cronix/internal/config"
	"github.com/QMZCLL/cronix/internal/cron"
	"github.com/QMZCLL/cronix/internal/task"
	"github.com/spf13/cobra"
)

func newDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <name>",
		Short: "Disable a scheduled task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if err := task.Disable(cfg.Tasks, name); err != nil {
				return err
			}

			cfg.Tasks = normalizeTasks(cfg.Tasks)
			if err := cron.SyncToCrontab(cfg.Tasks, wrappersDir()); err != nil {
				return err
			}
			if err := config.Save(cfg); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Task %q disabled\n", name)
			return nil
		},
	}
}
