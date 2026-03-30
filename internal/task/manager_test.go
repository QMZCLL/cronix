package task_test

import (
	"strings"
	"testing"

	"github.com/QMZCLL/cronix/internal/config"
	"github.com/QMZCLL/cronix/internal/task"
)

func TestAdd_Success(t *testing.T) {
	cfg := &config.Config{}
	scheduledTask := task.Task{
		Name:     "backup",
		Command:  "echo hi",
		CronExpr: "*/5 * * * *",
		Enabled:  true,
	}

	if err := task.Add(&cfg.Tasks, scheduledTask); err != nil {
		t.Fatalf("Add() unexpected error: %v", err)
	}

	if len(cfg.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(cfg.Tasks))
	}
	if cfg.Tasks[0].Name != scheduledTask.Name {
		t.Errorf("task name mismatch: want %q got %q", scheduledTask.Name, cfg.Tasks[0].Name)
	}
	if cfg.Tasks[0].CronExpr != scheduledTask.CronExpr {
		t.Errorf("cron expr mismatch: want %q got %q", scheduledTask.CronExpr, cfg.Tasks[0].CronExpr)
	}
	if cfg.Tasks[0].CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	found, err := task.FindByName(cfg.Tasks, scheduledTask.Name)
	if err != nil {
		t.Fatalf("FindByName() unexpected error: %v", err)
	}
	if found != &cfg.Tasks[0] {
		t.Fatal("FindByName() should return a pointer to the stored task")
	}

	listed := task.List(cfg.Tasks)
	if len(listed) != 1 {
		t.Fatalf("List() expected 1 task, got %d", len(listed))
	}
}

func TestAdd_DuplicateName(t *testing.T) {
	cfg := &config.Config{
		Tasks: []task.Task{{Name: "backup", CronExpr: "0 2 * * *", Command: "backup.sh"}},
	}

	err := task.Add(&cfg.Tasks, task.Task{Name: "backup", CronExpr: "*/5 * * * *", Command: "echo hi"})
	if err == nil {
		t.Fatal("expected duplicate name error")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected duplicate error message, got %q", err.Error())
	}
	if len(cfg.Tasks) != 1 {
		t.Fatalf("expected tasks to remain unchanged, got %d", len(cfg.Tasks))
	}
}

func TestAdd_InvalidCronExpr(t *testing.T) {
	cfg := &config.Config{}

	err := task.Add(&cfg.Tasks, task.Task{Name: "backup", CronExpr: "@daily", Command: "echo hi"})
	if err == nil {
		t.Fatal("expected invalid cron expression error")
	}
	if !strings.Contains(err.Error(), "invalid cron expression") {
		t.Fatalf("expected invalid cron expression error, got %q", err.Error())
	}
	if len(cfg.Tasks) != 0 {
		t.Fatalf("expected task not to be added, got %d", len(cfg.Tasks))
	}
}

func TestRemove_NotFound(t *testing.T) {
	cfg := &config.Config{}

	err := task.Remove(&cfg.Tasks, "missing")
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %q", err.Error())
	}
}

func TestEnable_Disable(t *testing.T) {
	cfg := &config.Config{
		Tasks: []task.Task{{Name: "backup", CronExpr: "0 2 * * *", Command: "backup.sh", Enabled: false}},
	}

	if err := task.Enable(cfg.Tasks, "backup"); err != nil {
		t.Fatalf("Enable() unexpected error: %v", err)
	}
	if !cfg.Tasks[0].Enabled {
		t.Fatal("expected task to be enabled")
	}

	found, err := task.FindByName(cfg.Tasks, "backup")
	if err != nil {
		t.Fatalf("FindByName() unexpected error after enable: %v", err)
	}
	if !found.Enabled {
		t.Fatal("expected found task to reflect enabled state")
	}

	if err := task.Disable(cfg.Tasks, "backup"); err != nil {
		t.Fatalf("Disable() unexpected error: %v", err)
	}
	if cfg.Tasks[0].Enabled {
		t.Fatal("expected task to be disabled")
	}
}

func TestRemove_Success(t *testing.T) {
	cfg := &config.Config{
		Tasks: []task.Task{
			{Name: "alpha", CronExpr: "0 1 * * *", Command: "echo alpha"},
			{Name: "beta", CronExpr: "0 2 * * *", Command: "echo beta"},
		},
	}

	if err := task.Remove(&cfg.Tasks, "alpha"); err != nil {
		t.Fatalf("Remove() unexpected error: %v", err)
	}
	if len(cfg.Tasks) != 1 {
		t.Fatalf("expected 1 task after remove, got %d", len(cfg.Tasks))
	}
	if cfg.Tasks[0].Name != "beta" {
		t.Errorf("expected remaining task to be 'beta', got %q", cfg.Tasks[0].Name)
	}
}

func TestRemove_SingleTask(t *testing.T) {
	cfg := &config.Config{
		Tasks: []task.Task{
			{Name: "only", CronExpr: "0 0 * * *", Command: "echo only"},
		},
	}

	if err := task.Remove(&cfg.Tasks, "only"); err != nil {
		t.Fatalf("Remove() unexpected error: %v", err)
	}
	if len(cfg.Tasks) != 0 {
		t.Fatalf("expected 0 tasks after remove, got %d", len(cfg.Tasks))
	}
}

func TestValidateCronPart_ValidCases(t *testing.T) {
	validExprs := []string{
		"0 0 * * *",
		"*/5 * * * *",
		"0-30 * * * *",
		"0 1-23 * * *",
		"0 0 1 1 0",
		"59 23 31 12 6",
	}
	for _, expr := range validExprs {
		cfg := &config.Config{}
		err := task.Add(&cfg.Tasks, task.Task{Name: "t", CronExpr: expr, Command: "echo"})
		if err != nil {
			t.Errorf("expected valid cron %q to succeed, got: %v", expr, err)
		}
	}
}

func TestValidateCronPart_InvalidCases(t *testing.T) {
	invalidExprs := []string{
		"60 * * * *",
		"* 24 * * *",
		"* * 32 * *",
		"* * * 13 *",

		"*/0 * * * *",
		"0-60 * * * *",
		"-1 * * * *",
		"abc * * * *",
		"@daily",
	}
	for _, expr := range invalidExprs {
		cfg := &config.Config{}
		err := task.Add(&cfg.Tasks, task.Task{Name: "t", CronExpr: expr, Command: "echo"})
		if err == nil {
			t.Errorf("expected invalid cron %q to fail", expr)
		}
	}
}
