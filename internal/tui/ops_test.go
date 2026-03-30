package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/QMZCLL/cronix/internal/config"
	"github.com/QMZCLL/cronix/internal/task"
)

func withTempCronixConfig(t *testing.T, tasks []task.Task) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CRONIX_CONFIG_DIR", dir)
	t.Setenv("CRONIX_LOG_DIR", filepath.Join(dir, "logs"))

	cfg := config.Config{Tasks: tasks, LogDir: filepath.Join(dir, "logs")}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), data, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func pressKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func TestOps_EnableKey_SetsStatusCmd(t *testing.T) {
	tasks := []task.Task{
		{Name: "backup", CronExpr: "0 * * * *", Enabled: false, Command: "echo hi"},
	}
	withTempCronixConfig(t, tasks)

	m := NewModel(tasks)
	updated, cmd := m.Update(pressKey('e'))
	_ = updated
	if cmd == nil {
		t.Fatal("expected a cmd from enable, got nil")
	}
}

func TestOps_DisableKey_SetsStatusCmd(t *testing.T) {
	tasks := []task.Task{
		{Name: "backup", CronExpr: "0 * * * *", Enabled: true, Command: "echo hi"},
	}
	withTempCronixConfig(t, tasks)

	m := NewModel(tasks)
	updated, cmd := m.Update(pressKey('d'))
	_ = updated
	if cmd == nil {
		t.Fatal("expected a cmd from disable, got nil")
	}
}

func TestOps_EnableOnEmptyList_NoOp(t *testing.T) {
	m := NewModel([]task.Task{})
	_, cmd := m.Update(pressKey('e'))
	if cmd != nil {
		t.Fatalf("expected nil cmd on empty list enable, got %#v", cmd)
	}
}

func TestOps_DisableOnEmptyList_NoOp(t *testing.T) {
	m := NewModel([]task.Task{})
	_, cmd := m.Update(pressKey('d'))
	if cmd != nil {
		t.Fatalf("expected nil cmd on empty list disable, got %#v", cmd)
	}
}

func TestOps_RunKey_ReturnsCmd(t *testing.T) {
	tasks := []task.Task{
		{Name: "echo-task", CronExpr: "0 * * * *", Enabled: true, Command: "echo hello"},
	}
	withTempCronixConfig(t, tasks)

	m := NewModel(tasks)
	_, cmd := m.Update(pressKey('r'))
	if cmd == nil {
		t.Fatal("expected a cmd from run key, got nil")
	}

	msg := cmd()
	rr, ok := msg.(runResult)
	if !ok {
		t.Fatalf("expected runResult msg, got %T", msg)
	}
	if rr.err != nil {
		t.Fatalf("expected successful run, got err: %v", rr.err)
	}
	if rr.exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", rr.exitCode)
	}
}

func TestOps_RunResult_ShowsStatusMessage(t *testing.T) {
	m := NewModel([]task.Task{{Name: "t", CronExpr: "0 * * * *", Command: "echo"}})

	updated, cmd := m.Update(runResult{exitCode: 0, duration: 0, err: nil})
	current := updated.(Model)

	if !strings.Contains(current.status, "Exit: 0") {
		t.Fatalf("expected status to contain 'Exit: 0', got %q", current.status)
	}
	if cmd == nil {
		t.Fatal("expected tick cmd after runResult, got nil")
	}
}

func TestOps_RunResult_Error_ShowsErrorStatus(t *testing.T) {
	m := NewModel([]task.Task{{Name: "t", CronExpr: "0 * * * *", Command: "echo"}})

	updated, cmd := m.Update(runResult{err: os.ErrNotExist})
	current := updated.(Model)

	if !current.statusErr {
		t.Fatal("expected statusErr=true on runResult error")
	}
	if cmd == nil {
		t.Fatal("expected tick cmd after error runResult, got nil")
	}
}

func TestOps_DeleteKey_ShowsConfirmPrompt(t *testing.T) {
	tasks := []task.Task{
		{Name: "backup", CronExpr: "0 * * * *", Enabled: true, Command: "echo"},
	}
	m := NewModel(tasks)

	updated, cmd := m.Update(pressKey('x'))
	current := updated.(Model)

	if cmd != nil {
		t.Fatalf("expected nil cmd from delete prompt, got %#v", cmd)
	}
	if !current.confirming {
		t.Fatal("expected confirming=true after x key")
	}
	if !strings.Contains(current.status, "backup") {
		t.Fatalf("expected status to contain task name, got %q", current.status)
	}
	if !strings.Contains(current.status, "[y/N]") {
		t.Fatalf("expected status to contain [y/N], got %q", current.status)
	}
}

func TestOps_DeleteKey_NAnswer_CancelsConfirm(t *testing.T) {
	tasks := []task.Task{
		{Name: "backup", CronExpr: "0 * * * *", Enabled: true, Command: "echo"},
	}
	m := NewModel(tasks)

	updated, _ := m.Update(pressKey('x'))
	current := updated.(Model)
	if !current.confirming {
		t.Fatal("expected confirming=true")
	}

	updated, cmd := current.Update(pressKey('n'))
	final := updated.(Model)

	if final.confirming {
		t.Fatal("expected confirming=false after n key")
	}
	if len(final.tasks) != 1 {
		t.Fatalf("expected task list unchanged (1), got %d", len(final.tasks))
	}
	if cmd != nil {
		t.Fatalf("expected nil cmd after cancel, got %#v", cmd)
	}
}

func TestOps_DeleteKey_YAnswer_CallsDelete(t *testing.T) {
	tasks := []task.Task{
		{Name: "backup", CronExpr: "0 * * * *", Enabled: true, Command: "echo"},
	}
	withTempCronixConfig(t, tasks)

	m := NewModel(tasks)

	updated, _ := m.Update(pressKey('x'))
	current := updated.(Model)

	updated, cmd := current.Update(pressKey('y'))
	final := updated.(Model)

	if final.confirming {
		t.Fatal("expected confirming=false after y")
	}
	if cmd == nil {
		t.Fatal("expected a cmd from delete confirmation, got nil")
	}
}

func TestOps_DeleteOnEmptyList_NoOp(t *testing.T) {
	m := NewModel([]task.Task{})
	_, cmd := m.Update(pressKey('x'))
	if cmd != nil {
		t.Fatalf("expected nil cmd on empty list delete, got %#v", cmd)
	}
}

func TestOps_ClearStatus_AfterTick(t *testing.T) {
	m := NewModel([]task.Task{{Name: "t", CronExpr: "0 * * * *", Command: "echo"}})
	m.status = "something"
	m.statusErr = true

	updated, _ := m.Update(clearStatusMsg{})
	current := updated.(Model)

	if current.status != "" {
		t.Fatalf("expected empty status after clearStatusMsg, got %q", current.status)
	}
	if current.statusErr {
		t.Fatal("expected statusErr=false after clearStatusMsg")
	}
}

func TestOps_HelpText_ContainsDelete(t *testing.T) {
	if !strings.Contains(helpText, "delete") && !strings.Contains(helpText, "[x]") {
		t.Fatalf("helpText should mention delete and x key, got: %q", helpText)
	}
}
