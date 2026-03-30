package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/QMZCLL/cronix/internal/task"
)

func withTempConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CRONIX_CONFIG_DIR", dir)
	return dir
}

func TestLoad_FileNotExists(t *testing.T) {
	withTempConfigDir(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil Config")
	}
	if len(cfg.Tasks) != 0 {
		t.Errorf("expected empty tasks, got %d", len(cfg.Tasks))
	}
}

func TestLoad_ValidFile(t *testing.T) {
	dir := withTempConfigDir(t)

	tasks := []task.Task{{Name: "backup", CronExpr: "@daily", Command: "rsync -a /src /dst"}}
	data, _ := json.Marshal(&Config{Tasks: tasks, LogDir: "/var/log/cronix"})
	if err := os.WriteFile(filepath.Join(dir, configFile), data, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(cfg.Tasks))
	}
	if cfg.Tasks[0].Name != "backup" {
		t.Errorf("expected task name 'backup', got %q", cfg.Tasks[0].Name)
	}
	if cfg.LogDir != "/var/log/cronix" {
		t.Errorf("expected log_dir '/var/log/cronix', got %q", cfg.LogDir)
	}
}

func TestSave_CreatesFile(t *testing.T) {
	dir := withTempConfigDir(t)

	cfg := &Config{Tasks: []task.Task{{Name: "hello", CronExpr: "* * * * *", Command: "echo hi"}}, LogDir: "/tmp/logs"}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, configFile)); err != nil {
		t.Errorf("tasks.json not created: %v", err)
	}
}

func TestSave_Atomic(t *testing.T) {
	dir := withTempConfigDir(t)

	cfg := &Config{Tasks: []task.Task{{Name: "atomic", CronExpr: "@hourly", Command: "true"}}}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != configFile {
			t.Errorf("unexpected leftover file after Save: %s", e.Name())
		}
	}
}

func TestConfig_RoundTrip(t *testing.T) {
	withTempConfigDir(t)

	lastRunAt := time.Date(2026, 3, 31, 6, 45, 0, 0, time.UTC)
	createdAt := time.Date(2026, 3, 30, 22, 15, 0, 0, time.UTC)
	orig := &Config{
		Tasks: []task.Task{
			{
				Name:        "ping",
				CronExpr:    "*/5 * * * *",
				Command:     "ping -c1 localhost",
				Enabled:     true,
				Envs:        map[string]string{"HOST": "localhost", "RETRIES": "3"},
				CreatedAt:   createdAt,
				LastRunAt:   &lastRunAt,
				Description: "connectivity health check",
			},
			{
				Name:        "clean",
				CronExpr:    "0 3 * * *",
				Command:     "rm -rf /tmp/old",
				Enabled:     false,
				Envs:        map[string]string{},
				CreatedAt:   createdAt.Add(2 * time.Hour),
				LastRunAt:   nil,
				Description: "cleanup temp files",
			},
		},
		LogDir: "/home/user/.local/share/cronix/logs",
	}

	if err := Save(orig); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.LogDir != orig.LogDir {
		t.Errorf("LogDir: want %q got %q", orig.LogDir, loaded.LogDir)
	}
	if !reflect.DeepEqual(loaded.Tasks, orig.Tasks) {
		t.Fatalf("Tasks mismatch\nwant: %#v\ngot:  %#v", orig.Tasks, loaded.Tasks)
	}
}

func TestEnsureConfigDir_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "newsubdir")
	t.Setenv("CRONIX_CONFIG_DIR", subdir)

	if err := EnsureConfigDir(); err != nil {
		t.Fatalf("EnsureConfigDir() unexpected error: %v", err)
	}

	info, err := os.Stat(subdir)
	if err != nil {
		t.Fatalf("expected directory to exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected a directory, got a file")
	}
}

func TestEnsureConfigDir_IdempotentIfExists(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CRONIX_CONFIG_DIR", dir)

	if err := EnsureConfigDir(); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := EnsureConfigDir(); err != nil {
		t.Fatalf("second call (idempotent): %v", err)
	}
}

func TestConfigDir_UsesEnvVar(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CRONIX_CONFIG_DIR", dir)

	got := ConfigDir()
	if got != dir {
		t.Errorf("ConfigDir() = %q, want %q", got, dir)
	}
}

func TestConfigDir_FallbackToHome(t *testing.T) {
	t.Setenv("CRONIX_CONFIG_DIR", "")

	got := ConfigDir()
	if got == "" {
		t.Fatal("ConfigDir() returned empty string")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := withTempConfigDir(t)

	if err := os.WriteFile(filepath.Join(dir, configFile), []byte("not-json{"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error on invalid JSON, got nil")
	}
}
