package cron

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const cronDaemonTestStateEnv = "CRONIX_TEST_CRON_DAEMON"

var procRoot = "/proc"

func CronDaemonWarning() string {
	running, known := cronDaemonRunning()
	if !known || running {
		return ""
	}
	return "Warning: cron daemon does not appear to be running; scheduled tasks will not fire until cron/crond is started."
}

func cronDaemonRunning() (running bool, known bool) {
	if forced, ok := forcedCronDaemonState(); ok {
		return forced, true
	}

	if runtime.GOOS != "linux" {
		return false, false
	}

	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return false, false
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		matched, err := processLooksLikeCron(entry.Name())
		if err != nil {
			continue
		}
		if matched {
			return true, true
		}
	}

	return false, true
}

func forcedCronDaemonState() (bool, bool) {
	state := strings.TrimSpace(strings.ToLower(os.Getenv(cronDaemonTestStateEnv)))
	switch state {
	case "running":
		return true, true
	case "stopped":
		return false, true
	case "", "auto":
		return false, false
	default:
		return false, false
	}
}

func processLooksLikeCron(pid string) (bool, error) {
	for _, name := range []string{"comm", "cmdline"} {
		content, err := os.ReadFile(filepath.Join(procRoot, pid, name))
		if err != nil {
			continue
		}
		if containsCronDaemonName(string(content)) {
			return true, nil
		}
	}
	return false, fmt.Errorf("cron process details unavailable for pid %s", pid)
}

func containsCronDaemonName(content string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(content), "\x00", " "))
	fields := strings.Fields(normalized)
	for _, field := range fields {
		base := filepath.Base(field)
		if base == "cron" || base == "crond" {
			return true
		}
	}
	return false
}
