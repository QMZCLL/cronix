package cron

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/QMZCLL/cronix/internal/task"
)

func WrapperPath(name, wrapperDir string) string {
	return filepath.Join(wrapperDir, name+".sh")
}

func GenerateWrapper(t task.Task, logDir string) string {
	var sb strings.Builder

	sb.WriteString("#!/usr/bin/env bash\n")
	sb.WriteString("set -euo pipefail\n")
	sb.WriteString("\n")

	if len(t.Envs) > 0 {
		keys := make([]string, 0, len(t.Envs))
		for k := range t.Envs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&sb, "export %s=%s\n", k, shellQuote(t.Envs[k]))
		}
		sb.WriteString("\n")
	}

	logPathExpr := fmt.Sprintf("%s/%s/$(date +%%Y-%%m-%%d).log", logDir, t.Name)
	sb.WriteString(fmt.Sprintf("LOG_FILE=\"%s\"\n", logPathExpr))
	sb.WriteString("mkdir -p \"$(dirname \"$LOG_FILE\")\"\n")
	sb.WriteString("\n")

	lockFile := fmt.Sprintf("/tmp/cronix-%s.lock", t.Name)
	sb.WriteString(fmt.Sprintf("LOCK_FILE=%s\n", shellQuote(lockFile)))
	sb.WriteString("\n")

	sb.WriteString("if [ -f \"$LOCK_FILE\" ]; then\n")
	sb.WriteString("    OLD_PID=$(cat \"$LOCK_FILE\")\n")
	sb.WriteString("    if kill -0 \"$OLD_PID\" 2>/dev/null; then\n")
	sb.WriteString("        echo \"[SKIPPED] $(date --iso-8601=seconds) previous run (PID $OLD_PID) still running\" >> \"$LOG_FILE\"\n")
	sb.WriteString("        exit 0\n")
	sb.WriteString("    fi\n")
	sb.WriteString("    rm -f \"$LOCK_FILE\"\n")
	sb.WriteString("fi\n")
	sb.WriteString("\n")

	sb.WriteString("echo $$ > \"$LOCK_FILE\"\n")
	sb.WriteString("trap 'rm -f \"$LOCK_FILE\"' EXIT\n")
	sb.WriteString("\n")

	sb.WriteString("echo \"=== Run at $(date --iso-8601=seconds) ===\" >> \"$LOG_FILE\"\n")
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("%s >> \"$LOG_FILE\" 2>&1\n", t.Command))
	sb.WriteString("EXIT_CODE=$?\n")
	sb.WriteString("\n")

	sb.WriteString("echo \"=== Exit: $EXIT_CODE ===\" >> \"$LOG_FILE\"\n")
	sb.WriteString("exit $EXIT_CODE\n")

	return sb.String()
}

func WriteWrapper(t task.Task, wrapperDir, logDir string) (string, error) {
	if err := os.MkdirAll(wrapperDir, 0o755); err != nil {
		return "", fmt.Errorf("cron: create wrapper dir %q: %w", wrapperDir, err)
	}
	path := WrapperPath(t.Name, wrapperDir)
	content := GenerateWrapper(t, logDir)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		return "", fmt.Errorf("cron: write wrapper %q: %w", path, err)
	}
	return path, nil
}

func RemoveWrapper(name, wrapperDir string) error {
	path := WrapperPath(name, wrapperDir)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cron: remove wrapper %q: %w", path, err)
	}
	return nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''")+"'"
}
