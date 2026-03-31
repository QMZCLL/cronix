package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/QMZCLL/cronix/internal/task"
)

func TestModelView_ShowsTasksAndHelp(t *testing.T) {
	model := NewModel([]task.Task{
		{Name: "backup", CronExpr: "0 * * * *", Enabled: true, Command: "echo short"},
		{Name: "trainer", CronExpr: "30 2 * * *", Enabled: false, Command: "python train.py --epochs 100 --dataset /very/long/path"},
	})

	view := model.View()

	for _, want := range []string{"NAME", "CRON", "STATUS", "COMMAND", "backup", "trainer", "enabled", "disabled", helpText} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected view to contain %q, got %q", want, view)
		}
	}

	if !strings.Contains(view, "python train.py --epoc...") {
		t.Fatalf("expected truncated command in view, got %q", view)
	}
}

func TestModelUpdate_AddPageNavigation(t *testing.T) {
	model := NewModel([]task.Task{{Name: "one"}})

	updated, cmd := model.Update(pressKey('a'))
	current := updated.(Model)
	if current.page != pageAdd {
		t.Fatalf("expected add page after a key, got %q", current.page)
	}
	if current.add.focus != addFieldName {
		t.Fatalf("expected focus on name field, got %d", current.add.focus)
	}
	if cmd == nil {
		t.Fatal("expected focus cmd when entering add page")
	}

	updated, _ = current.Update(tea.KeyMsg{Type: tea.KeyTab})
	current = updated.(Model)
	if current.add.focus != addFieldCommand {
		t.Fatalf("expected tab to move focus to command, got %d", current.add.focus)
	}

	updated, _ = current.Update(tea.KeyMsg{Type: tea.KeyEsc})
	current = updated.(Model)
	if current.page != pageList {
		t.Fatalf("expected esc to return to list, got %q", current.page)
	}
	if current.status != "" {
		t.Fatalf("expected status cleared when leaving add page, got %q", current.status)
	}
}

func TestRenderListView_ShowsOnceStatus(t *testing.T) {
	model := NewModel([]task.Task{{Name: "once-task", CronExpr: "0 * * * *", Enabled: true, RunOnce: true, Command: "echo once"}})

	view := model.View()
	if !strings.Contains(view, "once") {
		t.Fatalf("expected once status in view, got %q", view)
	}
}

func TestRenderListView_ShowsCurrentTime(t *testing.T) {
	model := NewModel([]task.Task{{Name: "clock-task", CronExpr: "0 * * * *", Enabled: true, Command: "echo now"}})
	model.width = 120
	model.now = time.Date(2026, 3, 31, 18, 7, 0, 0, time.Local)

	view := model.View()
	for _, want := range []string{"2026-03-31", "18:07"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected current date/time fragment %q in view, got %q", want, view)
		}
	}
}

func TestRenderListView_KeepsTomorrowNextRunOnSingleLine(t *testing.T) {
	model := NewModel([]task.Task{{Name: "backup", CronExpr: "38 17 * * *", Enabled: true, RunOnce: true, Command: "echo success"}})
	model.width = 140
	model.now = time.Date(2026, 3, 31, 18, 7, 0, 0, time.Local)

	view := model.View()
	if !strings.Contains(view, "tomorrow 17:38") {
		t.Fatalf("expected tomorrow next-run text in view, got %q", view)
	}
	if strings.Contains(view, "tomorrow\n") {
		t.Fatalf("expected next-run text not to wrap, got %q", view)
	}
	if strings.Contains(view, "17:38\n") {
		t.Fatalf("expected time fragment not to be forced onto a new line, got %q", view)
	}
}

func TestModelView_ShowsCronDaemonWarning(t *testing.T) {
	t.Setenv("CRONIX_TEST_CRON_DAEMON", "stopped")
	model := NewModel([]task.Task{{Name: "backup", CronExpr: "0 * * * *", Enabled: true, Command: "echo short"}})

	view := model.View()
	if !strings.Contains(view, "cron daemon does not appear to be running") {
		t.Fatalf("expected cron daemon warning in view, got %q", view)
	}
}

func TestModelUpdate_NavigationAndQuit(t *testing.T) {
	model := NewModel([]task.Task{{Name: "one"}, {Name: "two"}, {Name: "three"}})

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if cmd != nil {
		t.Fatalf("expected navigation to return nil cmd, got %#v", cmd)
	}
	current := updated.(Model)
	if current.cursor != 1 {
		t.Fatalf("expected cursor to move down to 1, got %d", current.cursor)
	}

	updated, _ = current.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	current = updated.(Model)
	if current.cursor != 0 {
		t.Fatalf("expected cursor to move up to 0, got %d", current.cursor)
	}

	_, cmd = current.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected ctrl+c to return a quit command")
	}
	_, cmd = current.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected q to return a quit command")
	}
}

func TestUpdateLogs_PrevDay_NoLog(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CRONIX_LOG_DIR", dir)

	model := NewModel([]task.Task{{Name: "sync"}})
	model.width = 80
	model.height = 24
	model.page = pageLogs
	model.logs = buildLogsState("sync", time.Now(), model.width, model.height)
	origDate := model.logs.date

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m := updated.(Model)
	if m.logs.date != origDate {
		t.Fatalf("expected date unchanged when no prev log, got %v", m.logs.date)
	}
}

func TestUpdateLogs_PrevDay_WithLog(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CRONIX_LOG_DIR", dir)

	taskName := "sync"
	today := time.Now()
	prev := today.AddDate(0, 0, -1)
	prevLogDir := filepath.Join(dir, taskName)
	if err := os.MkdirAll(prevLogDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	prevLogPath := filepath.Join(prevLogDir, prev.Format("2006-01-02")+".log")
	if err := os.WriteFile(prevLogPath, []byte("yesterday output\n"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	model := NewModel([]task.Task{{Name: taskName}})
	model.width = 80
	model.height = 24
	model.page = pageLogs
	model.logs = buildLogsState(taskName, today, model.width, model.height)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m := updated.(Model)
	if m.logs.date.Format("2006-01-02") != prev.Format("2006-01-02") {
		t.Fatalf("expected date to be prev day %s, got %v", prev.Format("2006-01-02"), m.logs.date)
	}
}

func TestExitCodeFromErr_Nil(t *testing.T) {
	if code := exitCodeFromErr(nil); code != 0 {
		t.Fatalf("expected 0 for nil err, got %d", code)
	}
}

func TestExitCodeFromErr_GenericError(t *testing.T) {
	err := fmt.Errorf("something failed")
	if code := exitCodeFromErr(err); code != 1 {
		t.Fatalf("expected 1 for generic error, got %d", code)
	}
}

func TestExitCodeFromErr_ExitError(t *testing.T) {
	cmd := exec.Command("sh", "-c", "exit 42")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	code := exitCodeFromErr(err)
	if code != 42 {
		t.Fatalf("expected exit code 42, got %d", code)
	}
}

func TestCommandEnv_NoOverrides(t *testing.T) {
	env := commandEnv(nil)
	if len(env) == 0 {
		t.Fatal("expected non-empty env")
	}
}

func TestCommandEnv_WithOverrides(t *testing.T) {
	t.Setenv("TEST_CRONIX_KEY", "original")
	overrides := map[string]string{"TEST_CRONIX_KEY": "overridden", "NEW_KEY": "newval"}
	env := commandEnv(overrides)

	var foundOverridden, foundNew bool
	var foundOriginal bool
	for _, e := range env {
		if e == "TEST_CRONIX_KEY=overridden" {
			foundOverridden = true
		}
		if e == "NEW_KEY=newval" {
			foundNew = true
		}
		if e == "TEST_CRONIX_KEY=original" {
			foundOriginal = true
		}
	}
	if !foundOverridden {
		t.Error("expected TEST_CRONIX_KEY=overridden")
	}
	if !foundNew {
		t.Error("expected NEW_KEY=newval")
	}
	if foundOriginal {
		t.Error("expected original TEST_CRONIX_KEY to be replaced")
	}
}

func TestHandleDeleteConfirmed_EmptyList(t *testing.T) {
	m := NewModel([]task.Task{})
	updated, cmd := m.Update(pressKey('x'))
	_ = updated
	if cmd != nil {
		t.Fatalf("expected nil cmd on empty list, got %#v", cmd)
	}
}
