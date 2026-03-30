package cron

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QMZCLL/cronix/internal/task"
)

func TestInjectBlock_Empty(t *testing.T) {
	block := strings.Join([]string{
		managedBlockStart,
		"*/5 * * * * /tmp/wrappers/backup.sh",
		managedBlockEnd,
	}, "\n")

	got := InjectBlock("", block)

	want := block + "\n"
	if got != want {
		t.Fatalf("InjectBlock() mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestInjectBlock_Replace(t *testing.T) {
	existing := strings.Join([]string{
		"MAILTO=user@example.com",
		managedBlockStart,
		"0 * * * * /old/wrapper.sh",
		managedBlockEnd,
		"@daily /usr/bin/backup-home",
	}, "\n")
	block := strings.Join([]string{
		managedBlockStart,
		"*/10 * * * * /new/wrapper.sh",
		managedBlockEnd,
	}, "\n")

	got := InjectBlock(existing, block)

	want := strings.Join([]string{
		"MAILTO=user@example.com",
		"",
		managedBlockStart,
		"*/10 * * * * /new/wrapper.sh",
		managedBlockEnd,
		"",
		"@daily /usr/bin/backup-home",
	}, "\n") + "\n"
	if got != want {
		t.Fatalf("InjectBlock() mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}

	again := InjectBlock(got, block)
	if again != want {
		t.Fatalf("InjectBlock() should be idempotent\nwant:\n%s\ngot:\n%s", want, again)
	}
}

func TestRemoveBlock(t *testing.T) {
	existing := strings.Join([]string{
		"SHELL=/bin/bash",
		managedBlockStart,
		"0 * * * * /tmp/one.sh",
		managedBlockEnd,
		"@daily /usr/local/bin/report",
	}, "\n")

	got := RemoveBlock(existing)

	want := strings.Join([]string{
		"SHELL=/bin/bash",
		"",
		"@daily /usr/local/bin/report",
	}, "\n")
	if got != want {
		t.Fatalf("RemoveBlock() mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestBuildEntry(t *testing.T) {
	scheduledTask := task.Task{Name: "backup", CronExpr: "*/15 * * * *"}

	got := BuildEntry(scheduledTask, "/tmp/cronix/backup.sh")

	want := "*/15 * * * * /tmp/cronix/backup.sh"
	if got != want {
		t.Fatalf("BuildEntry() = %q, want %q", got, want)
	}
}

func TestRead_NoCrontab(t *testing.T) {
	original := runCrontabCommand
	t.Cleanup(func() {
		runCrontabCommand = original
	})

	runCrontabCommand = func(stdin string, args ...string) (string, string, error) {
		if stdin != "" {
			t.Fatalf("Read should not pass stdin, got %q", stdin)
		}
		if len(args) != 1 || args[0] != "-l" {
			t.Fatalf("Read args mismatch: %v", args)
		}
		return "", "no crontab for qmz", errors.New("exit status 1")
	}

	got, err := Read()
	if err != nil {
		t.Fatalf("Read() unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("Read() = %q, want empty string", got)
	}
}

func TestSyncToCrontab_OnlyEnabledTasksManagedBlock(t *testing.T) {
	originalRead := readCrontab
	originalWrite := writeCrontab
	t.Cleanup(func() {
		readCrontab = originalRead
		writeCrontab = originalWrite
	})

	existing := strings.Join([]string{
		"MAILTO=ops@example.com",
		"@daily /usr/local/bin/external-backup",
		managedBlockStart,
		"0 * * * * /tmp/old-wrapper.sh",
		managedBlockEnd,
		"SHELL=/bin/sh",
	}, "\n")

	readCrontab = func() (string, error) {
		return existing, nil
	}

	var wrote string
	writeCrontab = func(content string) error {
		wrote = content
		return nil
	}

	tasks := []task.Task{
		{Name: "enabled-task", CronExpr: "*/5 * * * *", Enabled: true},
		{Name: "disabled-task", CronExpr: "0 0 * * *", Enabled: false},
	}

	wrapperDir := filepath.Join("/tmp", "cronix", "wrappers")
	if err := SyncToCrontab(tasks, wrapperDir); err != nil {
		t.Fatalf("SyncToCrontab() unexpected error: %v", err)
	}

	if !strings.Contains(wrote, "MAILTO=ops@example.com") {
		t.Fatalf("expected outside MAILTO line to be preserved, got:\n%s", wrote)
	}
	if !strings.Contains(wrote, "@daily /usr/local/bin/external-backup") {
		t.Fatalf("expected outside cron line to be preserved, got:\n%s", wrote)
	}
	if !strings.Contains(wrote, "SHELL=/bin/sh") {
		t.Fatalf("expected trailing outside line to be preserved, got:\n%s", wrote)
	}

	enabledEntry := BuildEntry(tasks[0], filepath.Join(wrapperDir, tasks[0].Name+".sh"))
	if !strings.Contains(wrote, enabledEntry) {
		t.Fatalf("expected enabled task entry %q in managed block, got:\n%s", enabledEntry, wrote)
	}
	disabledEntry := BuildEntry(tasks[1], filepath.Join(wrapperDir, tasks[1].Name+".sh"))
	if strings.Contains(wrote, disabledEntry) {
		t.Fatalf("did not expect disabled task entry %q in managed block, got:\n%s", disabledEntry, wrote)
	}
	if strings.Count(wrote, managedBlockStart) != 1 || strings.Count(wrote, managedBlockEnd) != 1 {
		t.Fatalf("expected exactly one managed block, got:\n%s", wrote)
	}
}
