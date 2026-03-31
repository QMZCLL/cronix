package cron

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCronDaemonWarning(t *testing.T) {
	t.Setenv(cronDaemonTestStateEnv, "")
	originalProcRoot := procRoot
	t.Cleanup(func() {
		procRoot = originalProcRoot
	})

	t.Run("running returns no warning", func(t *testing.T) {
		t.Setenv(cronDaemonTestStateEnv, "running")
		if got := CronDaemonWarning(); got != "" {
			t.Fatalf("CronDaemonWarning() = %q, want empty string", got)
		}
	})

	t.Run("stopped returns warning", func(t *testing.T) {
		t.Setenv(cronDaemonTestStateEnv, "stopped")
		got := CronDaemonWarning()
		if got == "" {
			t.Fatal("expected warning when cron daemon is stopped")
		}
	})

	t.Run("proc scan detects cron process", func(t *testing.T) {
		t.Setenv(cronDaemonTestStateEnv, "auto")
		procRoot = t.TempDir()
		writeProcEntry(t, procRoot, "123", "cron\n", "")

		if got := CronDaemonWarning(); got != "" {
			t.Fatalf("CronDaemonWarning() = %q, want empty string", got)
		}
	})

	t.Run("proc scan without cron warns", func(t *testing.T) {
		t.Setenv(cronDaemonTestStateEnv, "auto")
		procRoot = t.TempDir()
		writeProcEntry(t, procRoot, "456", "bash\n", "/usr/bin/bash\x00-lc\x00echo hi")

		got := CronDaemonWarning()
		if got == "" {
			t.Fatal("expected warning when proc scan cannot find cron")
		}
	})

	t.Run("unavailable proc root degrades gracefully", func(t *testing.T) {
		t.Setenv(cronDaemonTestStateEnv, "auto")
		procRoot = filepath.Join(t.TempDir(), "missing-proc")

		if got := CronDaemonWarning(); got != "" {
			t.Fatalf("CronDaemonWarning() = %q, want empty string when detection unavailable", got)
		}
	})
}

func writeProcEntry(t *testing.T, root, pid, comm, cmdline string) {
	t.Helper()
	dir := filepath.Join(root, pid)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir proc entry: %v", err)
	}
	if comm != "" {
		if err := os.WriteFile(filepath.Join(dir, "comm"), []byte(comm), 0o644); err != nil {
			t.Fatalf("write comm: %v", err)
		}
	}
	if cmdline != "" {
		if err := os.WriteFile(filepath.Join(dir, "cmdline"), []byte(cmdline), 0o644); err != nil {
			t.Fatalf("write cmdline: %v", err)
		}
	}
}
