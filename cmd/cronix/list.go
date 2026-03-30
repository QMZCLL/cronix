package main

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/QMZCLL/cronix/internal/config"
	"github.com/QMZCLL/cronix/internal/task"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List scheduled tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			tasks := normalizeTasks(cfg.Tasks)
			if jsonOutput {
				return writeTasksJSON(cmd, tasks)
			}

			writeTasksTable(cmd, tasks)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output tasks as JSON")
	return cmd
}

func writeTasksJSON(cmd *cobra.Command, tasks []task.Task) error {
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tasks json: %w", err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", data)
	return nil
}

func writeTasksTable(cmd *cobra.Command, tasks []task.Task) {
	tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "NAME\tCRON\tSTATUS\tCOMMAND")
	for _, scheduledTask := range tasks {
		status := "disabled"
		if scheduledTask.Enabled {
			status = "enabled"
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", scheduledTask.Name, scheduledTask.CronExpr, status, scheduledTask.Command)
	}
	_ = tw.Flush()
}
