package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/QMZCLL/cronix/internal/config"
	"github.com/QMZCLL/cronix/internal/logger"
	"github.com/QMZCLL/cronix/internal/task"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run <name>",
		Short: "Run a task immediately",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			scheduledTask, err := task.FindByName(cfg.Tasks, args[0])
			if err != nil {
				return err
			}

			logDir := logger.LogDir(cfg.LogDir)
			logFile, err := logger.OpenLogFile(logDir, scheduledTask.Name)
			if err != nil {
				return err
			}
			defer logFile.Close()

			startedAt := time.Now()
			logger.WriteHeader(logFile, startedAt)

			runner := exec.Command("bash", "-lc", scheduledTask.Command)
			runner.Env = commandEnv(scheduledTask.Envs)
			runner.Stdout = io.MultiWriter(cmd.OutOrStdout(), logFile)
			runner.Stderr = io.MultiWriter(cmd.ErrOrStderr(), logFile)

			runErr := runner.Run()
			exitCode := exitCodeFromError(runErr)
			duration := time.Since(startedAt)
			logger.WriteFooter(logFile, exitCode, duration)

			logPath := logger.LogPath(logDir, scheduledTask.Name, startedAt)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Exit: %d | Duration: %s | Log: %s\n", exitCode, duration.Round(time.Millisecond), logPath)

			if runErr != nil {
				return fmt.Errorf("run task %q: %w", scheduledTask.Name, runErr)
			}
			return nil
		},
	}
}

func commandEnv(taskEnvs map[string]string) []string {
	env := os.Environ()
	if len(taskEnvs) == 0 {
		return env
	}

	filtered := make([]string, 0, len(env)+len(taskEnvs))
	for _, entry := range env {
		key, _, ok := strings.Cut(entry, "=")
		if ok {
			if _, exists := taskEnvs[key]; exists {
				continue
			}
		}
		filtered = append(filtered, entry)
	}

	for key, value := range taskEnvs {
		filtered = append(filtered, key+"="+value)
	}

	return filtered
}

func exitCodeFromError(err error) int {
	if err == nil {
		return 0
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}

	return 1
}
