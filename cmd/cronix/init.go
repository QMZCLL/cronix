package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/QMZCLL/cronix/internal/config"
	"github.com/QMZCLL/cronix/internal/cron"
	"github.com/QMZCLL/cronix/internal/task"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize cronix configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.EnsureConfigDir(); err != nil {
				return err
			}

			configPath := filepath.Join(config.ConfigDir(), "tasks.json")
			if _, err := os.Stat(configPath); err != nil {
				if !os.IsNotExist(err) {
					return fmt.Errorf("stat config %q: %w", configPath, err)
				}
				if err := config.Save(&config.Config{Tasks: []task.Task{}}); err != nil {
					return err
				}
			}

			existing, err := cron.Read()
			if err != nil {
				return err
			}
			managed := cron.InjectBlock(existing, cron.EmptyManagedBlock())
			if err := cron.Write(managed); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Initialized cronix at %s\n", config.ConfigDir())
			return nil
		},
	}
}
