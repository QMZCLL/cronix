package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogPath(t *testing.T) {
	given := struct {
		logDir   string
		taskName string
		t        time.Time
	}{
		logDir:   "/var/log/cronix",
		taskName: "backup",
		t:        time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC),
	}

	when := LogPath(given.logDir, given.taskName, given.t)

	want := "/var/log/cronix/backup/2026-03-30.log"
	if when != want {
		t.Errorf("got %q, want %q", when, want)
	}
}

func TestOpenLogFile_CreatesDir(t *testing.T) {
	given := t.TempDir()
	taskName := "mytask"

	when, err := OpenLogFile(given, taskName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	when.Close()

	expectedDir := filepath.Join(given, taskName)
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("expected dir %q to exist", expectedDir)
	}

	today := time.Now().Format("2006-01-02")
	expectedFile := filepath.Join(expectedDir, today+".log")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("expected file %q to exist", expectedFile)
	}
}

func TestWriteHeader(t *testing.T) {
	given := time.Date(2026, 3, 30, 9, 0, 0, 0, time.UTC)
	var buf bytes.Buffer

	WriteHeader(&buf, given)

	when := buf.String()
	if !strings.HasPrefix(when, "=== Run at") {
		t.Errorf("header should start with '=== Run at', got: %q", when)
	}
	if !strings.Contains(when, "2026-03-30T09:00:00Z") {
		t.Errorf("header should contain RFC3339 timestamp, got: %q", when)
	}
}

func TestWriteFooter(t *testing.T) {
	var buf bytes.Buffer

	WriteFooter(&buf, 0, 2*time.Second+500*time.Millisecond)

	when := buf.String()
	if !strings.HasPrefix(when, "=== Exit:") {
		t.Errorf("footer should start with '=== Exit:', got: %q", when)
	}
	if !strings.Contains(when, "2.5s") {
		t.Errorf("footer should contain duration, got: %q", when)
	}
}

func TestLogDir_EnvOverride(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CRONIX_LOG_DIR", tmp)

	when := LogDir("some-other-dir")

	if when != tmp {
		t.Errorf("env override: got %q, want %q", when, tmp)
	}
}

func TestLogDir_CfgFallback(t *testing.T) {
	t.Setenv("CRONIX_LOG_DIR", "")
	tmp := t.TempDir()

	when := LogDir(tmp)

	if when != tmp {
		t.Errorf("cfg fallback: got %q, want %q", when, tmp)
	}
}

func TestLogDir_Default(t *testing.T) {
	t.Setenv("CRONIX_LOG_DIR", "")

	when := LogDir("")

	if !strings.HasSuffix(when, "cronix-logs") {
		t.Errorf("default: expected suffix 'cronix-logs', got %q", when)
	}
}
