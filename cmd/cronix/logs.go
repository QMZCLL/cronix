package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/QMZCLL/cronix/internal/config"
	"github.com/QMZCLL/cronix/internal/logger"
	"github.com/spf13/cobra"
)

func newLogsCmd() *cobra.Command {
	var (
		dateValue      string
		tailLinesCount int
	)

	cmd := &cobra.Command{
		Use:   "logs <name>",
		Short: "Show task logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			logDate, err := resolveLogDate(dateValue)
			if err != nil {
				return err
			}

			logPath := logger.LogPath(logger.LogDir(cfg.LogDir), args[0], logDate)
			data, err := os.ReadFile(logPath)
			if err != nil {
				return fmt.Errorf("read log %q: %w", logPath, err)
			}

			content := string(data)
			if tailLinesCount > 0 {
				content = tailLines(content, tailLinesCount)
			}

			if content != "" {
				_, _ = fmt.Fprint(cmd.OutOrStdout(), content)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dateValue, "date", "", "Log date in YYYY-MM-DD format")
	cmd.Flags().IntVar(&tailLinesCount, "tail", 0, "Show only the last N lines")

	return cmd
}

func resolveLogDate(value string) (time.Time, error) {
	if value == "" {
		return time.Now(), nil
	}

	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --date %q: expected YYYY-MM-DD", value)
	}
	return parsed, nil
}

func tailLines(content string, n int) string {
	if n <= 0 || content == "" {
		return content
	}

	hasTrailingNewline := strings.HasSuffix(content, "\n")
	trimmed := strings.TrimRight(content, "\n")
	if trimmed == "" {
		return content
	}

	lines := strings.Split(trimmed, "\n")
	if n >= len(lines) {
		return content
	}

	result := strings.Join(lines[len(lines)-n:], "\n")
	if hasTrailingNewline {
		result += "\n"
	}
	return result
}
