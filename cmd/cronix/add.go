package main

import (
	"fmt"
	"strings"

	"github.com/QMZCLL/cronix/internal/config"
	"github.com/QMZCLL/cronix/internal/cron"
	"github.com/QMZCLL/cronix/internal/logger"
	"github.com/QMZCLL/cronix/internal/task"
	"github.com/spf13/cobra"
)

type envFlag []string

func (e *envFlag) String() string {
	return strings.Join(*e, ",")
}

func (e *envFlag) Set(value string) error {
	*e = append(*e, value)
	return nil
}

func (e *envFlag) Type() string {
	return "env"
}

func newAddCmd() *cobra.Command {
	var (
		name        string
		cronExpr    string
		command     string
		description string
		runOnce     bool
		envs        envFlag
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a scheduled task",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("cron") {
				return fmt.Errorf("required flag \"cron\" not set")
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			taskEnvs, err := task.ParseEnvAssignments(envs)
			if err != nil {
				return err
			}

			scheduledTask := task.Task{
				Name:        name,
				CronExpr:    cronExpr,
				Command:     command,
				Description: description,
				Enabled:     true,
				RunOnce:     runOnce,
				Envs:        taskEnvs,
			}

			if err := task.Add(&cfg.Tasks, scheduledTask); err != nil {
				return err
			}

			logDir := logger.LogDir(cfg.LogDir)
			if _, err := cron.WriteWrapper(scheduledTask, wrappersDir(), logDir); err != nil {
				return err
			}

			cfg.Tasks = normalizeTasks(cfg.Tasks)
			if err := cron.SyncToCrontab(cfg.Tasks, wrappersDir()); err != nil {
				return err
			}
			if err := config.Save(cfg); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Task %q added (runs: %s)\n", scheduledTask.Name, scheduledTask.CronExpr)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&cronExpr, "cron", "", "Cron expression")
	cmd.Flags().StringVar(&command, "cmd", "", "Command to run")
	cmd.Flags().StringVar(&description, "desc", "", "Task description")
	cmd.Flags().BoolVar(&runOnce, "once", false, "Run task on its cron schedule once, then auto-disable after the first successful run")
	cmd.Flags().Var(&envs, "env", "Environment variable (KEY=VALUE), may be repeated")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("cmd")

	return cmd
}
