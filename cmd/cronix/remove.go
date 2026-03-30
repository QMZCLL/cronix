package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/QMZCLL/cronix/internal/config"
	"github.com/QMZCLL/cronix/internal/cron"
	"github.com/QMZCLL/cronix/internal/task"
	"github.com/spf13/cobra"
)

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a scheduled task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			confirmed, err := confirmRemoval(cmd, name)
			if err != nil {
				return err
			}
			if !confirmed {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
				return nil
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := task.Remove(&cfg.Tasks, name); err != nil {
				return err
			}

			if err := cron.RemoveWrapper(name, wrappersDir()); err != nil {
				return err
			}

			cfg.Tasks = normalizeTasks(cfg.Tasks)
			if err := cron.SyncToCrontab(cfg.Tasks, wrappersDir()); err != nil {
				return err
			}
			if err := config.Save(cfg); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Task %q removed\n", name)
			return nil
		},
	}
}

func confirmRemoval(cmd *cobra.Command, name string) (bool, error) {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Remove task %q? [y/N] ", name); err != nil {
		return false, err
	}

	reader := bufio.NewReader(cmd.InOrStdin())
	response, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}

	answer := strings.TrimSpace(strings.ToLower(response))
	return answer == "y" || answer == "yes", nil
}
