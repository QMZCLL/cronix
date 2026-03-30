package cron

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QMZCLL/cronix/internal/task"
)

func TestFullWorkflow(t *testing.T) {
	t.Run("#given a new enabled task and fake crontab seams", func(t *testing.T) {
		originalRead := readCrontab
		originalWrite := writeCrontab
		t.Cleanup(func() {
			readCrontab = originalRead
			writeCrontab = originalWrite
		})

		wrapperDir := filepath.Join(t.TempDir(), "wrappers")
		logDir := filepath.Join(t.TempDir(), "logs")
		existingCrontab := strings.Join([]string{
			"SHELL=/bin/bash",
			"MAILTO=ops@example.com",
		}, "\n")

		readCrontab = func() (string, error) {
			return existingCrontab, nil
		}

		var wrote string
		writeCrontab = func(content string) error {
			wrote = content
			return nil
		}

		var tasks []task.Task
		scheduledTask := task.Task{
			Name:        "backup",
			CronExpr:    "*/5 * * * *",
			Command:     "echo backup",
			Enabled:     true,
			Envs:        map[string]string{"APP_ENV": "prod"},
			Description: "nightly backup",
		}

		t.Run("#when the task is added written as a wrapper and synced", func(t *testing.T) {
			if err := task.Add(&tasks, scheduledTask); err != nil {
				t.Fatalf("Add() unexpected error: %v", err)
			}
			if len(tasks) != 1 {
				t.Fatalf("expected 1 task after Add(), got %d", len(tasks))
			}
			if tasks[0].CreatedAt.IsZero() {
				t.Fatal("expected Add() to populate CreatedAt")
			}

			wrapperPath, err := WriteWrapper(tasks[0], wrapperDir, logDir)
			if err != nil {
				t.Fatalf("WriteWrapper() unexpected error: %v", err)
			}

			if err := SyncToCrontab(tasks, wrapperDir); err != nil {
				t.Fatalf("SyncToCrontab() unexpected error: %v", err)
			}

			t.Run("#then the wrapper file is written without touching a real crontab", func(t *testing.T) {
				contents, err := os.ReadFile(wrapperPath)
				if err != nil {
					t.Fatalf("read wrapper file: %v", err)
				}
				if !strings.Contains(string(contents), "echo backup") {
					t.Fatalf("expected wrapper to contain task command, got:\n%s", contents)
				}
				if !strings.Contains(string(contents), "export APP_ENV='prod'") {
					t.Fatalf("expected wrapper to contain exported env, got:\n%s", contents)
				}
			})

			t.Run("#then the managed crontab block contains the generated wrapper path", func(t *testing.T) {
				expectedEntry := BuildEntry(tasks[0], wrapperPath)
				if !strings.Contains(wrote, expectedEntry) {
					t.Fatalf("expected crontab to contain %q, got:\n%s", expectedEntry, wrote)
				}
				if strings.Count(wrote, managedBlockStart) != 1 || strings.Count(wrote, managedBlockEnd) != 1 {
					t.Fatalf("expected exactly one managed block, got:\n%s", wrote)
				}
			})

			t.Run("#then unmanaged crontab lines are preserved", func(t *testing.T) {
				if !strings.Contains(wrote, "SHELL=/bin/bash") {
					t.Fatalf("expected SHELL line to be preserved, got:\n%s", wrote)
				}
				if !strings.Contains(wrote, "MAILTO=ops@example.com") {
					t.Fatalf("expected MAILTO line to be preserved, got:\n%s", wrote)
				}
			})
		})
	})
}
