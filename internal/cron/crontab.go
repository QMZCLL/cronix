package cron

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/QMZCLL/cronix/internal/task"
)

const (
	managedBlockStart = "# cronix-managed-start"
	managedBlockEnd   = "# cronix-managed-end"
)

var (
	runCrontabCommand = defaultRunCrontabCommand
	readCrontab       = Read
	writeCrontab      = Write
)

func Read() (string, error) {
	stdout, stderr, err := runCrontabCommand("", "-l")
	if err != nil {
		if isNoCrontabError(stdout, stderr) {
			return "", nil
		}
		return "", fmt.Errorf("cron: read crontab: %w", err)
	}
	return strings.TrimRight(stdout, "\n"), nil
}

func Write(content string) error {
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if _, _, err := runCrontabCommand(content, "-"); err != nil {
		return fmt.Errorf("cron: write crontab: %w", err)
	}
	return nil
}

func InjectBlock(existing, block string) string {
	managed := strings.TrimRight(block, "\n")
	if managed == "" {
		return strings.TrimRight(RemoveBlock(existing), "\n")
	}

	normalized := normalizeNewlines(existing)
	startIdx := strings.Index(normalized, managedBlockStart)
	if startIdx == -1 {
		cleaned := strings.TrimRight(RemoveBlock(existing), "\n")
		if cleaned == "" {
			return managed + "\n"
		}
		return cleaned + "\n\n" + managed + "\n"
	}

	endSearchStart := startIdx + len(managedBlockStart)
	endRelIdx := strings.Index(normalized[endSearchStart:], managedBlockEnd)
	if endRelIdx == -1 {
		cleaned := strings.TrimRight(RemoveBlock(existing), "\n")
		if cleaned == "" {
			return managed + "\n"
		}
		return cleaned + "\n\n" + managed + "\n"
	}

	endIdx := endSearchStart + endRelIdx + len(managedBlockEnd)
	if endIdx < len(normalized) && normalized[endIdx] == '\n' {
		endIdx++
	}

	before := strings.TrimRight(normalized[:startIdx], "\n")
	after := strings.TrimLeft(normalized[endIdx:], "\n")

	switch {
	case before == "" && after == "":
		return managed + "\n"
	case before == "":
		return ensureTrailingNewline(managed + "\n\n" + strings.TrimRight(after, "\n"))
	case after == "":
		return before + "\n\n" + managed + "\n"
	default:
		return ensureTrailingNewline(before + "\n\n" + managed + "\n\n" + strings.TrimRight(after, "\n"))
	}
}

func RemoveBlock(existing string) string {
	normalized := normalizeNewlines(existing)
	startIdx := strings.Index(normalized, managedBlockStart)
	if startIdx == -1 {
		return strings.TrimRight(normalized, "\n")
	}

	endSearchStart := startIdx + len(managedBlockStart)
	endRelIdx := strings.Index(normalized[endSearchStart:], managedBlockEnd)
	if endRelIdx == -1 {
		return strings.TrimRight(normalized[:startIdx], "\n")
	}

	endIdx := endSearchStart + endRelIdx + len(managedBlockEnd)
	if endIdx < len(normalized) && normalized[endIdx] == '\n' {
		endIdx++
	}

	before := strings.TrimRight(normalized[:startIdx], "\n")
	after := strings.TrimLeft(normalized[endIdx:], "\n")

	switch {
	case before == "":
		return strings.TrimRight(after, "\n")
	case after == "":
		return before
	default:
		return before + "\n\n" + strings.TrimRight(after, "\n")
	}
}

func BuildEntry(t task.Task, wrapperPath string) string {
	return t.CronExpr + " " + wrapperPath
}

func EmptyManagedBlock() string {
	return strings.Join([]string{managedBlockStart, managedBlockEnd}, "\n")
}

func SyncToCrontab(tasks []task.Task, wrapperDir string) error {
	existing, err := readCrontab()
	if err != nil {
		return err
	}

	block := buildManagedBlock(tasks, wrapperDir)
	updated := RemoveBlock(existing)
	if block != "" {
		updated = InjectBlock(existing, block)
	}

	if updated == strings.TrimRight(normalizeNewlines(existing), "\n") {
		return nil
	}

	if err := writeCrontab(updated); err != nil {
		return err
	}
	return nil
}

func buildManagedBlock(tasks []task.Task, wrapperDir string) string {
	lines := []string{managedBlockStart}
	for _, scheduledTask := range tasks {
		if !scheduledTask.Enabled {
			continue
		}
		wrapperPath := filepath.Join(wrapperDir, scheduledTask.Name+".sh")
		lines = append(lines, BuildEntry(scheduledTask, wrapperPath))
	}
	if len(lines) == 1 {
		return ""
	}
	lines = append(lines, managedBlockEnd)
	return strings.Join(lines, "\n")
}

func defaultRunCrontabCommand(stdin string, args ...string) (string, string, error) {
	cmd := exec.Command("crontab", args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func isNoCrontabError(stdout, stderr string) bool {
	combined := strings.ToLower(strings.TrimSpace(stdout + "\n" + stderr))
	return strings.Contains(combined, "no crontab for")
}

func normalizeNewlines(content string) string {
	return strings.ReplaceAll(content, "\r\n", "\n")
}

func ensureTrailingNewline(content string) string {
	if content == "" || strings.HasSuffix(content, "\n") {
		return content
	}
	return content + "\n"
}
