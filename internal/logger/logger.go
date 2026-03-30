package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func LogDir(cfgLogDir string) string {
	if v := os.Getenv("CRONIX_LOG_DIR"); v != "" {
		return expandHome(v)
	}
	if cfgLogDir != "" {
		return expandHome(cfgLogDir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, "cronix-logs")
}

func LogPath(logDir, taskName string, t time.Time) string {
	date := t.Format("2006-01-02")
	return filepath.Join(logDir, taskName, date+".log")
}

func OpenLogFile(logDir, taskName string) (*os.File, error) {
	path := LogPath(logDir, taskName, time.Now())
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("logger: create log dir %q: %w", dir, err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("logger: open log file %q: %w", path, err)
	}
	return f, nil
}

func WriteHeader(w io.Writer, startTime time.Time) {
	fmt.Fprintf(w, "=== Run at %s ===\n", startTime.Format(time.RFC3339))
}

func WriteFooter(w io.Writer, exitCode int, duration time.Duration) {
	fmt.Fprintf(w, "=== Exit: %d | Duration: %s ===\n", exitCode, duration.Round(time.Millisecond))
}

func expandHome(p string) string {
	if len(p) >= 2 && p[0] == '~' && p[1] == '/' {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		return filepath.Join(home, p[2:])
	}
	return p
}
