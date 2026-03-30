package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/QMZCLL/cronix/internal/task"
)

func TestLogsView_ShowsContent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CRONIX_LOG_DIR", dir)

	taskName := "backup"
	date := time.Now()
	logPath := filepath.Join(dir, taskName, date.Format("2006-01-02")+".log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(logPath, []byte("job output line\n"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	model := NewModel([]task.Task{{Name: taskName}})
	model.width = 120
	model.height = 40
	model.logs = buildLogsState(taskName, date, model.width, model.height)
	model.page = pageLogs

	view := model.View()

	if !strings.Contains(view, "job output line") {
		t.Fatalf("expected log content in view, got:\n%s", view)
	}
	if !strings.Contains(view, taskName) {
		t.Fatalf("expected task name %q in header, got:\n%s", taskName, view)
	}
	if !strings.Contains(view, date.Format("2006-01-02")) {
		t.Fatalf("expected date in header, got:\n%s", view)
	}
}

func TestLogsView_NoLog(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CRONIX_LOG_DIR", dir)

	taskName := "trainer"
	date := time.Now()

	model := NewModel([]task.Task{{Name: taskName}})
	model.width = 120
	model.height = 40
	model.logs = buildLogsState(taskName, date, model.width, model.height)
	model.page = pageLogs

	view := model.View()

	wantPhrase := "No logs for today"
	if !strings.Contains(view, wantPhrase) {
		t.Fatalf("expected placeholder %q in view, got:\n%s", wantPhrase, view)
	}
}

func TestLogsView_BackNavigation(t *testing.T) {
	taskName := "sync"
	dir := t.TempDir()
	t.Setenv("CRONIX_LOG_DIR", dir)

	model := NewModel([]task.Task{{Name: taskName}})
	model.width = 120
	model.height = 40
	model.logs = buildLogsState(taskName, time.Now(), model.width, model.height)
	model.page = pageLogs

	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'b'}},
	} {
		updated, _ := model.Update(key)
		m := updated.(Model)
		if m.page != pageList {
			t.Fatalf("expected page=pageList after 'b', got %q", m.page)
		}
	}

	model.page = pageLogs
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m := updated.(Model)
	if m.page != pageList {
		t.Fatalf("expected page=pageList after esc, got %q", m.page)
	}
}
