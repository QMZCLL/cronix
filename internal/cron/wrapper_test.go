package cron

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QMZCLL/cronix/internal/task"
)

func TestGenerateWrapper_BasicScript(t *testing.T) {
	t.Run("#given a task with no envs", func(t *testing.T) {
		tsk := task.Task{
			Name:    "backup",
			Command: "/usr/local/bin/backup.sh",
		}
		logDir := "/var/log/cronix"

		t.Run("#when GenerateWrapper is called", func(t *testing.T) {
			got := GenerateWrapper(tsk, logDir)

			t.Run("#then script starts with shebang and set -euo pipefail", func(t *testing.T) {
				if !strings.HasPrefix(got, "#!/usr/bin/env bash\n") {
					t.Fatalf("expected shebang line, got:\n%s", got)
				}
				if !strings.Contains(got, "set -euo pipefail") {
					t.Fatalf("expected set -euo pipefail, got:\n%s", got)
				}
			})

			t.Run("#then log path contains task name and date expression", func(t *testing.T) {
				expected := "/var/log/cronix/backup/$(date +%Y-%m-%d).log"
				if !strings.Contains(got, expected) {
					t.Fatalf("expected log path %q in script, got:\n%s", expected, got)
				}
			})

			t.Run("#then log path is double-quoted so date expands at runtime", func(t *testing.T) {
				if !strings.Contains(got, `LOG_FILE="/var/log/cronix/backup/$(date +%Y-%m-%d).log"`) {
					t.Fatalf("expected LOG_FILE to be double-quoted for runtime expansion, got:\n%s", got)
				}
			})

			t.Run("#then command is present", func(t *testing.T) {
				if !strings.Contains(got, "/usr/local/bin/backup.sh") {
					t.Fatalf("expected command in script, got:\n%s", got)
				}
			})

			t.Run("#then lockfile path contains task name", func(t *testing.T) {
				if !strings.Contains(got, "/tmp/cronix-backup.lock") {
					t.Fatalf("expected lockfile in script, got:\n%s", got)
				}
			})

			t.Run("#then trap cleanup is present", func(t *testing.T) {
				if !strings.Contains(got, "trap") || !strings.Contains(got, "EXIT") {
					t.Fatalf("expected trap ... EXIT in script, got:\n%s", got)
				}
			})
		})
	})
}

func TestGenerateWrapper_WithEnvVars(t *testing.T) {
	t.Run("#given a task with environment variables", func(t *testing.T) {
		tsk := task.Task{
			Name:    "etl",
			Command: "python etl.py",
			Envs: map[string]string{
				"DB_HOST": "localhost",
				"API_KEY": "secret",
			},
		}

		t.Run("#when GenerateWrapper is called", func(t *testing.T) {
			got := GenerateWrapper(tsk, "/logs")

			t.Run("#then all env vars are exported", func(t *testing.T) {
				if !strings.Contains(got, "export API_KEY=") {
					t.Fatalf("expected export API_KEY in script, got:\n%s", got)
				}
				if !strings.Contains(got, "export DB_HOST=") {
					t.Fatalf("expected export DB_HOST in script, got:\n%s", got)
				}
			})

			t.Run("#then env vars appear before the command", func(t *testing.T) {
				exportIdx := strings.Index(got, "export ")
				cmdIdx := strings.Index(got, "python etl.py")
				if exportIdx == -1 || cmdIdx == -1 || exportIdx > cmdIdx {
					t.Fatalf("expected export lines before command, got:\n%s", got)
				}
			})

			t.Run("#then values are single-quoted", func(t *testing.T) {
				if !strings.Contains(got, "'localhost'") {
					t.Fatalf("expected single-quoted value 'localhost', got:\n%s", got)
				}
			})
		})
	})
}

func TestGenerateWrapper_LockfileSkip(t *testing.T) {
	t.Run("#given a generated wrapper script", func(t *testing.T) {
		tsk := task.Task{Name: "report", Command: "./report.sh"}
		got := GenerateWrapper(tsk, "/logs")

		t.Run("#when checking skip logic", func(t *testing.T) {
			t.Run("#then script contains [SKIPPED] output line", func(t *testing.T) {
				if !strings.Contains(got, "[SKIPPED]") {
					t.Fatalf("expected [SKIPPED] in lockfile skip block, got:\n%s", got)
				}
			})

			t.Run("#then script uses kill -0 to check running PID", func(t *testing.T) {
				if !strings.Contains(got, "kill -0") {
					t.Fatalf("expected kill -0 in lockfile check, got:\n%s", got)
				}
			})

			t.Run("#then stale lock is removed before proceeding", func(t *testing.T) {
				if !strings.Contains(got, "rm -f \"$LOCK_FILE\"") {
					t.Fatalf("expected rm -f lock file for stale cleanup, got:\n%s", got)
				}
			})
		})
	})
}

func TestWriteWrapper_Executable(t *testing.T) {
	t.Run("#given a temp wrapper dir and a task", func(t *testing.T) {
		wrapperDir := t.TempDir()
		logDir := t.TempDir()
		tsk := task.Task{Name: "nightly", Command: "echo nightly"}

		t.Run("#when WriteWrapper is called", func(t *testing.T) {
			path, err := WriteWrapper(tsk, wrapperDir, logDir)
			if err != nil {
				t.Fatalf("WriteWrapper() unexpected error: %v", err)
			}

			t.Run("#then returned path matches WrapperPath", func(t *testing.T) {
				want := WrapperPath(tsk.Name, wrapperDir)
				if path != want {
					t.Fatalf("WriteWrapper() path = %q, want %q", path, want)
				}
			})

			t.Run("#then file exists at expected path", func(t *testing.T) {
				expected := filepath.Join(wrapperDir, "nightly.sh")
				if _, err := os.Stat(expected); err != nil {
					t.Fatalf("expected wrapper file at %q: %v", expected, err)
				}
			})

			t.Run("#then file is executable", func(t *testing.T) {
				info, err := os.Stat(filepath.Join(wrapperDir, "nightly.sh"))
				if err != nil {
					t.Fatalf("stat wrapper file: %v", err)
				}
				if info.Mode()&0o111 == 0 {
					t.Fatalf("expected wrapper file to be executable, mode=%v", info.Mode())
				}
			})

			t.Run("#then file content contains task command", func(t *testing.T) {
				contents, err := os.ReadFile(filepath.Join(wrapperDir, "nightly.sh"))
				if err != nil {
					t.Fatalf("read wrapper file: %v", err)
				}
				if !strings.Contains(string(contents), "echo nightly") {
					t.Fatalf("expected command in wrapper file, got:\n%s", contents)
				}
			})

			t.Run("#then RemoveWrapper deletes the file", func(t *testing.T) {
				if err := RemoveWrapper(tsk.Name, wrapperDir); err != nil {
					t.Fatalf("RemoveWrapper() unexpected error: %v", err)
				}
				if _, err := os.Stat(filepath.Join(wrapperDir, "nightly.sh")); !os.IsNotExist(err) {
					t.Fatalf("expected wrapper file to be removed")
				}
			})

			t.Run("#then RemoveWrapper on missing file returns nil", func(t *testing.T) {
				if err := RemoveWrapper("nonexistent", wrapperDir); err != nil {
					t.Fatalf("RemoveWrapper() on missing file should return nil, got: %v", err)
				}
			})
		})
	})
}

func TestRunWrapper_CreatesDateLog(t *testing.T) {
	t.Run("#given a wrapper written to a temp dir", func(t *testing.T) {
		wrapperDir := t.TempDir()
		logDir := t.TempDir()
		tsk := task.Task{
			Name:    "datelog",
			Command: "echo hello",
			Enabled: true,
		}
		path, err := WriteWrapper(tsk, wrapperDir, logDir)
		if err != nil {
			t.Fatalf("WriteWrapper() unexpected error: %v", err)
		}

		t.Run("#when the wrapper is executed", func(t *testing.T) {
			cmd := exec.Command("bash", path)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("wrapper execution failed: %v\noutput: %s", err, out)
			}

			t.Run("#then a log file is created under logDir/taskName/YYYY-MM-DD.log", func(t *testing.T) {
				date := time.Now().Format("2006-01-02")
				expectedLog := filepath.Join(logDir, "datelog", date+".log")
				if _, err := os.Stat(expectedLog); os.IsNotExist(err) {
					t.Fatalf("expected log file at %q to exist after wrapper run", expectedLog)
				}
			})

			t.Run("#then the log file contains the command output", func(t *testing.T) {
				date := time.Now().Format("2006-01-02")
				logPath := filepath.Join(logDir, "datelog", date+".log")
				contents, err := os.ReadFile(logPath)
				if err != nil {
					t.Fatalf("read log file: %v", err)
				}
				if !strings.Contains(string(contents), "hello") {
					t.Fatalf("expected 'hello' in log file, got:\n%s", contents)
				}
			})
		})
	})
}
