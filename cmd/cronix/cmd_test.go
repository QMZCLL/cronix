package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QMZCLL/cronix/internal/config"
	"github.com/QMZCLL/cronix/internal/logger"
	"github.com/QMZCLL/cronix/internal/task"
)

func TestCmdInit_Success(t *testing.T) {
	state := setupCLIEnv(t)

	stdout, stderr, err := executeCLI(t, nil, "init")
	if err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "Initialized cronix") {
		t.Fatalf("expected init success output, got %q", stdout)
	}

	configPath := filepath.Join(state.configDir, "tasks.json")
	data, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatalf("expected tasks.json to exist: %v", readErr)
	}
	if strings.Contains(string(data), `"tasks": null`) {
		t.Fatalf("expected empty tasks array, got %s", string(data))
	}

	crontabContent := readFakeCrontabState(t, state)
	if !strings.Contains(crontabContent, "# cronix-managed-start") || !strings.Contains(crontabContent, "# cronix-managed-end") {
		t.Fatalf("expected managed block in crontab, got:\n%s", crontabContent)
	}
	if strings.Contains(crontabContent, ".sh") {
		t.Fatalf("did not expect task entries in init crontab block, got:\n%s", crontabContent)
	}
}

func TestCmdAdd_Success(t *testing.T) {
	state := setupCLIEnv(t)

	stdout, stderr, err := executeCLI(t, nil,
		"add",
		"--name", "backup",
		"--cron", "0 * * * *",
		"--cmd", "echo hello",
	)
	if err != nil {
		t.Fatalf("add failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, `✓ Task "backup" added (runs: 0 * * * *)`) {
		t.Fatalf("unexpected success output: %q", stdout)
	}

	cfg := loadConfigForTest(t)
	if len(cfg.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(cfg.Tasks))
	}
	if cfg.Tasks[0].Name != "backup" {
		t.Fatalf("expected task backup, got %q", cfg.Tasks[0].Name)
	}
	if !cfg.Tasks[0].Enabled {
		t.Fatal("expected added task to be enabled")
	}

	wrapperPath := filepath.Join(state.configDir, "wrappers", "backup.sh")
	if _, statErr := os.Stat(wrapperPath); statErr != nil {
		t.Fatalf("expected wrapper file to exist: %v", statErr)
	}

	crontabContent := readFakeCrontabState(t, state)
	if !strings.Contains(crontabContent, wrapperPath) {
		t.Fatalf("expected wrapper path in crontab, got:\n%s", crontabContent)
	}
}

func TestCmdAdd_DuplicateName(t *testing.T) {
	setupCLIEnv(t)
	seedConfig(t, &config.Config{Tasks: []task.Task{{
		Name:     "backup",
		CronExpr: "0 * * * *",
		Command:  "echo first",
		Enabled:  true,
	}}})

	stdout, stderr, err := executeCLI(t, nil,
		"add",
		"--name", "backup",
		"--cron", "*/5 * * * *",
		"--cmd", "echo second",
	)
	if err == nil {
		t.Fatalf("expected duplicate add to fail, stdout=%q stderr=%q", stdout, stderr)
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected duplicate error, got %v", err)
	}

	cfg := loadConfigForTest(t)
	if len(cfg.Tasks) != 1 {
		t.Fatalf("expected config to remain unchanged, got %d tasks", len(cfg.Tasks))
	}
}

func TestCmdAdd_WithEnvVars(t *testing.T) {
	setupCLIEnv(t)

	_, stderr, err := executeCLI(t, nil,
		"add",
		"--name", "trainer",
		"--cron", "*/15 * * * *",
		"--cmd", "python train.py",
		"--env", "CUDA_VISIBLE_DEVICES=0",
		"--env", "PYTHONPATH=/opt/ml",
	)
	if err != nil {
		t.Fatalf("add with envs failed: %v\nstderr: %s", err, stderr)
	}

	cfg := loadConfigForTest(t)
	if len(cfg.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(cfg.Tasks))
	}

	got := cfg.Tasks[0].Envs
	if got["CUDA_VISIBLE_DEVICES"] != "0" {
		t.Fatalf("expected CUDA_VISIBLE_DEVICES env, got %#v", got)
	}
	if got["PYTHONPATH"] != "/opt/ml" {
		t.Fatalf("expected PYTHONPATH env, got %#v", got)
	}
}

func TestCmdAdd_OnceUsesProvidedCronSchedule(t *testing.T) {
	setupCLIEnv(t)

	stdout, stderr, err := executeCLI(t, nil,
		"add",
		"--name", "once-backup",
		"--cron", "5 4 * * 1",
		"--cmd", "echo once",
		"--once",
	)
	if err != nil {
		t.Fatalf("add with once failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, `✓ Task "once-backup" added (runs: 5 4 * * 1)`) {
		t.Fatalf("unexpected success output: %q", stdout)
	}

	cfg := loadConfigForTest(t)
	if len(cfg.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(cfg.Tasks))
	}
	if cfg.Tasks[0].CronExpr != "5 4 * * 1" {
		t.Fatalf("expected configured cron to be preserved, got %q", cfg.Tasks[0].CronExpr)
	}
	if !cfg.Tasks[0].RunOnce {
		t.Fatal("expected run_once to be persisted")
	}
}

func TestCmdAdd_OnceStillRequiresCron(t *testing.T) {
	setupCLIEnv(t)

	_, stderr, err := executeCLI(t, nil,
		"add",
		"--name", "once-backup",
		"--cmd", "echo once",
		"--once",
	)
	if err == nil {
		t.Fatalf("expected add without cron to fail, stderr=%q", stderr)
	}
	if !strings.Contains(err.Error(), `required flag "cron" not set`) {
		t.Fatalf("expected missing cron error, got %v", err)
	}
}

func TestCmdList_Empty(t *testing.T) {
	setupCLIEnv(t)

	stdout, stderr, err := executeCLI(t, nil, "list")
	if err != nil {
		t.Fatalf("list failed: %v\nstderr: %s", err, stderr)
	}

	trimmed := strings.TrimSpace(stdout)
	lines := strings.Split(trimmed, "\n")
	if len(lines) != 1 {
		t.Fatalf("expected only table header for empty list, got %q", stdout)
	}
	if !strings.Contains(lines[0], "NAME") || !strings.Contains(lines[0], "CRON") || !strings.Contains(lines[0], "STATUS") || !strings.Contains(lines[0], "COMMAND") {
		t.Fatalf("expected table header, got %q", lines[0])
	}
}

func TestCmdList_WithTasks(t *testing.T) {
	setupCLIEnv(t)
	seedConfig(t, &config.Config{Tasks: []task.Task{
		{Name: "backup", CronExpr: "0 * * * *", Command: "echo backup", Enabled: true},
		{Name: "cleanup", CronExpr: "30 2 * * *", Command: "echo clean", Enabled: false},
	}})

	stdout, stderr, err := executeCLI(t, nil, "list")
	if err != nil {
		t.Fatalf("list failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "backup") || !strings.Contains(stdout, "enabled") {
		t.Fatalf("expected enabled task row, got %q", stdout)
	}
	if !strings.Contains(stdout, "cleanup") || !strings.Contains(stdout, "disabled") {
		t.Fatalf("expected disabled task row, got %q", stdout)
	}

	jsonOut, jsonErrOut, err := executeCLI(t, nil, "list", "--json")
	if err != nil {
		t.Fatalf("list --json failed: %v\nstderr: %s", err, jsonErrOut)
	}

	var got []task.Task
	if unmarshalErr := json.Unmarshal([]byte(jsonOut), &got); unmarshalErr != nil {
		t.Fatalf("expected valid json output, got error %v with payload %q", unmarshalErr, jsonOut)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 tasks in json output, got %d", len(got))
	}
}

func TestCmdRemove_WithConfirm(t *testing.T) {
	state := setupCLIEnv(t)
	seedConfig(t, &config.Config{Tasks: []task.Task{{
		Name:     "backup",
		CronExpr: "0 * * * *",
		Command:  "echo backup",
		Enabled:  true,
	}}})

	wrapperDir := filepath.Join(state.configDir, "wrappers")
	if err := os.MkdirAll(wrapperDir, 0o755); err != nil {
		t.Fatalf("setup wrapper dir: %v", err)
	}
	wrapperPath := filepath.Join(wrapperDir, "backup.sh")
	if err := os.WriteFile(wrapperPath, []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatalf("setup wrapper file: %v", err)
	}
	writeFakeCrontabState(t, state, strings.Join([]string{
		"MAILTO=ops@example.com",
		"# cronix-managed-start",
		"0 * * * * " + wrapperPath,
		"# cronix-managed-end",
	}, "\n")+"\n")

	stdout, stderr, err := executeCLI(t, strings.NewReader("y\n"), "remove", "backup")
	if err != nil {
		t.Fatalf("remove failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, `Remove task "backup"? [y/N]`) {
		t.Fatalf("expected confirmation prompt, got %q", stdout)
	}
	if !strings.Contains(stdout, `✓ Task "backup" removed`) {
		t.Fatalf("expected remove success output, got %q", stdout)
	}

	cfg := loadConfigForTest(t)
	if len(cfg.Tasks) != 0 {
		t.Fatalf("expected no tasks after remove, got %d", len(cfg.Tasks))
	}
	if _, statErr := os.Stat(wrapperPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected wrapper removed, stat err=%v", statErr)
	}

	crontabContent := readFakeCrontabState(t, state)
	if strings.Contains(crontabContent, wrapperPath) {
		t.Fatalf("expected wrapper entry removed from crontab, got:\n%s", crontabContent)
	}
	if !strings.Contains(crontabContent, "MAILTO=ops@example.com") {
		t.Fatalf("expected external crontab content preserved, got:\n%s", crontabContent)
	}
}

func TestCmdEnable_Success(t *testing.T) {
	state := setupCLIEnv(t)
	seedConfig(t, &config.Config{Tasks: []task.Task{{
		Name:     "backup",
		CronExpr: "0 * * * *",
		Command:  "echo backup",
		Enabled:  false,
	}}})

	stdout, stderr, err := executeCLI(t, nil, "enable", "backup")
	if err != nil {
		t.Fatalf("enable failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, `✓ Task "backup" enabled`) {
		t.Fatalf("unexpected enable output: %q", stdout)
	}

	cfg := loadConfigForTest(t)
	if len(cfg.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(cfg.Tasks))
	}
	if !cfg.Tasks[0].Enabled {
		t.Fatal("expected task to be enabled")
	}

	wrapperPath := filepath.Join(state.configDir, "wrappers", "backup.sh")
	if _, statErr := os.Stat(wrapperPath); statErr != nil {
		t.Fatalf("expected wrapper file to exist after enable: %v", statErr)
	}

	crontabContent := readFakeCrontabState(t, state)
	if !strings.Contains(crontabContent, wrapperPath) {
		t.Fatalf("expected crontab to include wrapper path, got:\n%s", crontabContent)
	}
}

func TestCmdDisable_Success(t *testing.T) {
	state := setupCLIEnv(t)
	seedConfig(t, &config.Config{Tasks: []task.Task{{
		Name:     "backup",
		CronExpr: "0 * * * *",
		Command:  "echo backup",
		Enabled:  true,
	}}})

	wrapperDir := filepath.Join(state.configDir, "wrappers")
	if err := os.MkdirAll(wrapperDir, 0o755); err != nil {
		t.Fatalf("setup wrapper dir: %v", err)
	}
	wrapperPath := filepath.Join(wrapperDir, "backup.sh")
	if err := os.WriteFile(wrapperPath, []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatalf("setup wrapper file: %v", err)
	}
	writeFakeCrontabState(t, state, strings.Join([]string{
		"SHELL=/bin/bash",
		"# cronix-managed-start",
		"0 * * * * " + wrapperPath,
		"# cronix-managed-end",
	}, "\n")+"\n")

	stdout, stderr, err := executeCLI(t, nil, "disable", "backup")
	if err != nil {
		t.Fatalf("disable failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, `✓ Task "backup" disabled`) {
		t.Fatalf("unexpected disable output: %q", stdout)
	}

	cfg := loadConfigForTest(t)
	if len(cfg.Tasks) != 1 {
		t.Fatalf("expected 1 task to remain, got %d", len(cfg.Tasks))
	}
	if cfg.Tasks[0].Enabled {
		t.Fatal("expected task to be disabled")
	}

	if _, statErr := os.Stat(wrapperPath); statErr != nil {
		t.Fatalf("expected wrapper to remain on disk, got %v", statErr)
	}

	crontabContent := readFakeCrontabState(t, state)
	if strings.Contains(crontabContent, wrapperPath) {
		t.Fatalf("expected crontab entry removed, got:\n%s", crontabContent)
	}
	if !strings.Contains(crontabContent, "SHELL=/bin/bash") {
		t.Fatalf("expected unmanaged lines preserved, got:\n%s", crontabContent)
	}
}

func TestCmdRun_StreamsAndWritesLog(t *testing.T) {
	setupCLIEnv(t)
	logDir := t.TempDir()
	seedConfig(t, &config.Config{Tasks: []task.Task{{
		Name:     "backup",
		CronExpr: "0 * * * *",
		Command:  `printf 'hello from stdout\n'; printf 'hello from stderr\n' >&2`,
		Enabled:  true,
	}}, LogDir: logDir})

	stdout, stderr, err := executeCLI(t, nil, "run", "backup")
	if err != nil {
		t.Fatalf("run failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "hello from stdout") {
		t.Fatalf("expected stdout stream content, got %q", stdout)
	}
	if !strings.Contains(stderr, "hello from stderr") {
		t.Fatalf("expected stderr stream content, got %q", stderr)
	}
	stdoutLines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	if len(stdoutLines) != 2 {
		t.Fatalf("expected streamed stdout plus single summary line, got %q", stdout)
	}
	if stdoutLines[0] != "hello from stdout" {
		t.Fatalf("expected streamed stdout line first, got %q", stdoutLines[0])
	}
	if !strings.HasPrefix(stdoutLines[1], "✓ Exit: 0 | Duration: ") {
		t.Fatalf("expected single-line success summary, got %q", stdoutLines[1])
	}
	if !strings.Contains(stdoutLines[1], " | Log: ") {
		t.Fatalf("expected log segment in summary, got %q", stdoutLines[1])
	}

	logPath := logger.LogPath(logDir, "backup", time.Now())
	data, readErr := os.ReadFile(logPath)
	if readErr != nil {
		t.Fatalf("expected log file to exist: %v", readErr)
	}
	logContent := string(data)
	if !strings.Contains(logContent, "=== Run at") {
		t.Fatalf("expected log header, got %q", logContent)
	}
	if !strings.Contains(logContent, "hello from stdout") || !strings.Contains(logContent, "hello from stderr") {
		t.Fatalf("expected combined output in log, got %q", logContent)
	}
	if !strings.Contains(logContent, "=== Exit: 0 | Duration:") {
		t.Fatalf("expected log footer, got %q", logContent)
	}
	if !strings.Contains(stdoutLines[1], logPath) {
		t.Fatalf("expected summary to reference log path %q, got %q", logPath, stdoutLines[1])
	}
}

func TestCmdLogs_DateAndTail(t *testing.T) {
	setupCLIEnv(t)
	logDir := t.TempDir()
	seedConfig(t, &config.Config{LogDir: logDir})

	logDate := time.Date(2026, 3, 31, 8, 0, 0, 0, time.UTC)
	logPath := logger.LogPath(logDir, "backup", logDate)
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatalf("create log dir: %v", err)
	}
	content := strings.Join([]string{"line 1", "line 2", "line 3", "line 4"}, "\n") + "\n"
	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write log file: %v", err)
	}

	stdout, stderr, err := executeCLI(t, nil, "logs", "backup", "--date", "2026-03-31", "--tail", "2")
	if err != nil {
		t.Fatalf("logs failed: %v\nstderr: %s", err, stderr)
	}
	if stdout != "line 3\nline 4\n" {
		t.Fatalf("unexpected tailed logs output: %q", stdout)
	}
}

func TestCmdLogs_InvalidDate(t *testing.T) {
	setupCLIEnv(t)

	stdout, stderr, err := executeCLI(t, nil, "logs", "backup", "--date", "20260331")
	if err == nil {
		t.Fatalf("expected invalid date error, stdout=%q stderr=%q", stdout, stderr)
	}
	if !strings.Contains(err.Error(), "expected YYYY-MM-DD") {
		t.Fatalf("unexpected invalid date error: %v", err)
	}
}

type cliTestState struct {
	configDir    string
	binDir       string
	crontabState string
}

func setupCLIEnv(t *testing.T) cliTestState {
	t.Helper()

	configDir := t.TempDir()
	binDir := t.TempDir()
	state := cliTestState{
		configDir:    configDir,
		binDir:       binDir,
		crontabState: filepath.Join(t.TempDir(), "crontab.txt"),
	}

	t.Setenv("CRONIX_CONFIG_DIR", configDir)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	writeFakeCrontabBinary(t, state)
	writeFakeCrontabState(t, state, "")

	return state
}

func writeFakeCrontabBinary(t *testing.T, state cliTestState) {
	t.Helper()

	script := strings.Join([]string{
		"#!/usr/bin/env bash",
		"set -euo pipefail",
		"STATE_FILE=${CRONIX_TEST_CRONTAB_STATE:?}",
		"if [ \"$#\" -eq 1 ] && [ \"$1\" = \"-l\" ]; then",
		"  if [ ! -f \"$STATE_FILE\" ] || [ ! -s \"$STATE_FILE\" ]; then",
		"    echo \"no crontab for test\" >&2",
		"    exit 1",
		"  fi",
		"  cat \"$STATE_FILE\"",
		"  exit 0",
		"fi",
		"if [ \"$#\" -eq 1 ] && [ \"$1\" = \"-\" ]; then",
		"  cat > \"$STATE_FILE\"",
		"  exit 0",
		"fi",
		"echo \"unsupported args: $*\" >&2",
		"exit 2",
	}, "\n") + "\n"

	path := filepath.Join(state.binDir, "crontab")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake crontab binary: %v", err)
	}
	t.Setenv("CRONIX_TEST_CRONTAB_STATE", state.crontabState)
}

func writeFakeCrontabState(t *testing.T, state cliTestState, content string) {
	t.Helper()
	if err := os.WriteFile(state.crontabState, []byte(content), 0o644); err != nil {
		t.Fatalf("write fake crontab state: %v", err)
	}
}

func readFakeCrontabState(t *testing.T, state cliTestState) string {
	t.Helper()
	data, err := os.ReadFile(state.crontabState)
	if err != nil {
		t.Fatalf("read fake crontab state: %v", err)
	}
	return string(data)
}

func seedConfig(t *testing.T, cfg *config.Config) {
	t.Helper()
	if err := config.Save(cfg); err != nil {
		t.Fatalf("seed config: %v", err)
	}
}

func loadConfigForTest(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return cfg
}

func executeCLI(t *testing.T, stdin io.Reader, args ...string) (string, string, error) {
	t.Helper()

	cmd := newRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	if stdin != nil {
		cmd.SetIn(stdin)
	}
	cmd.SetArgs(args)

	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}
