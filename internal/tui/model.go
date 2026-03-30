package tui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/QMZCLL/cronix/internal/config"
	"github.com/QMZCLL/cronix/internal/cron"
	"github.com/QMZCLL/cronix/internal/logger"
	"github.com/QMZCLL/cronix/internal/task"
)

type page string

const (
	pageList page = "list"
	pageLogs page = "logs"

	statusClearDelay = 3 * time.Second
)

type logsState struct {
	taskName string
	date     time.Time
	viewport viewport.Model
	content  string
}

type statusMsg struct {
	text    string
	isError bool
}

type clearStatusMsg struct{}

type runResult struct {
	exitCode int
	duration time.Duration
	err      error
}

type Model struct {
	page       page
	tasks      []task.Task
	cursor     int
	width      int
	height     int
	logs       logsState
	status     string
	statusErr  bool
	confirming bool
}

func NewModel(tasks []task.Task) Model {
	model := Model{
		page:  pageList,
		tasks: normalizeTasks(tasks),
	}

	if len(model.tasks) == 0 {
		model.cursor = 0
	}

	return model
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		if m.page == pageLogs {
			m.logs = buildLogsState(m.logs.taskName, m.logs.date, m.width, m.height)
		}
		return m, nil
	case statusMsg:
		m.status = typed.text
		m.statusErr = typed.isError
		return m, scheduleStatusClear()
	case clearStatusMsg:
		m.status = ""
		m.statusErr = false
		return m, nil
	case runResult:
		if typed.err != nil {
			m.status = fmt.Sprintf("Error: %s", typed.err.Error())
			m.statusErr = true
		} else {
			m.status = fmt.Sprintf("Exit: %d | Duration: %s", typed.exitCode, typed.duration.Round(time.Millisecond))
			m.statusErr = false
		}
		return m, scheduleStatusClear()
	case tea.KeyMsg:
		switch m.page {
		case pageList:
			return m.updateList(typed)
		case pageLogs:
			return m.updateLogs(typed)
		}
	}

	return m, nil
}

func (m Model) updateList(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.confirming {
		return m.updateConfirm(key)
	}

	switch key.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		m.moveCursor(-1)
	case "down", "j":
		m.moveCursor(1)
	case "l":
		if len(m.tasks) > 0 {
			name := m.tasks[m.cursor].Name
			m.logs = buildLogsState(name, time.Now(), m.width, m.height)
			m.page = pageLogs
		}
	case "e":
		return m.handleEnable()
	case "d":
		return m.handleDisable()
	case "r":
		return m.handleRun()
	case "x", "delete":
		return m.handleDeletePrompt()
	}
	return m, nil
}

func (m Model) updateConfirm(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.confirming = false
	m.status = ""
	m.statusErr = false
	if key.String() == "y" {
		return m.handleDeleteConfirmed()
	}
	return m, nil
}

func (m Model) updateLogs(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "b", "esc":
		m.page = pageList
		return m, nil
	case "p":
		prevDate := m.logs.date.AddDate(0, 0, -1)
		prevLogs := buildLogsState(m.logs.taskName, prevDate, m.width, m.height)
		if prevLogs.content != "" {
			m.logs = prevLogs
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.logs.viewport, cmd = m.logs.viewport.Update(key)
	return m, cmd
}

func (m Model) View() string {
	switch m.page {
	case pageLogs:
		return renderLogsView(m)
	default:
		return renderListView(m)
	}
}

func (m *Model) moveCursor(delta int) {
	if len(m.tasks) == 0 {
		m.cursor = 0
		return
	}

	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.tasks) {
		m.cursor = len(m.tasks) - 1
	}
}

func normalizeTasks(tasks []task.Task) []task.Task {
	if tasks == nil {
		return []task.Task{}
	}
	return tasks
}

func (m Model) handleEnable() (tea.Model, tea.Cmd) {
	if len(m.tasks) == 0 {
		return m, nil
	}
	name := m.tasks[m.cursor].Name
	cfg, err := config.Load()
	if err != nil {
		return m, sendError(fmt.Sprintf("load config: %s", err))
	}
	if err := task.Enable(cfg.Tasks, name); err != nil {
		return m, sendError(fmt.Sprintf("enable: %s", err))
	}
	wrapperDir := filepath.Join(config.ConfigDir(), "wrappers")
	if err := cron.SyncToCrontab(cfg.Tasks, wrapperDir); err != nil {
		return m, sendError(fmt.Sprintf("sync crontab: %s", err))
	}
	if err := config.Save(cfg); err != nil {
		return m, sendError(fmt.Sprintf("save config: %s", err))
	}
	m.tasks = cfg.Tasks
	return m, sendStatus(fmt.Sprintf("Enabled %q", name))
}

func (m Model) handleDisable() (tea.Model, tea.Cmd) {
	if len(m.tasks) == 0 {
		return m, nil
	}
	name := m.tasks[m.cursor].Name
	cfg, err := config.Load()
	if err != nil {
		return m, sendError(fmt.Sprintf("load config: %s", err))
	}
	if err := task.Disable(cfg.Tasks, name); err != nil {
		return m, sendError(fmt.Sprintf("disable: %s", err))
	}
	wrapperDir := filepath.Join(config.ConfigDir(), "wrappers")
	if err := cron.SyncToCrontab(cfg.Tasks, wrapperDir); err != nil {
		return m, sendError(fmt.Sprintf("sync crontab: %s", err))
	}
	if err := config.Save(cfg); err != nil {
		return m, sendError(fmt.Sprintf("save config: %s", err))
	}
	m.tasks = cfg.Tasks
	return m, sendStatus(fmt.Sprintf("Disabled %q", name))
}

func (m Model) handleRun() (tea.Model, tea.Cmd) {
	if len(m.tasks) == 0 {
		return m, nil
	}
	scheduledTask := m.tasks[m.cursor]
	return m, func() tea.Msg {
		cfg, err := config.Load()
		if err != nil {
			return runResult{err: err}
		}
		logDir := logger.LogDir(cfg.LogDir)
		logFile, err := logger.OpenLogFile(logDir, scheduledTask.Name)
		if err != nil {
			return runResult{err: err}
		}
		defer logFile.Close()

		startedAt := time.Now()
		logger.WriteHeader(logFile, startedAt)

		runner := exec.Command("bash", "-lc", scheduledTask.Command)
		runner.Env = commandEnv(scheduledTask.Envs)
		runner.Stdout = io.MultiWriter(os.Stdout, logFile)
		runner.Stderr = io.MultiWriter(os.Stderr, logFile)

		runErr := runner.Run()
		exitCode := exitCodeFromErr(runErr)
		duration := time.Since(startedAt)
		logger.WriteFooter(logFile, exitCode, duration)

		return runResult{exitCode: exitCode, duration: duration, err: runErr}
	}
}

func (m Model) handleDeletePrompt() (tea.Model, tea.Cmd) {
	if len(m.tasks) == 0 {
		return m, nil
	}
	name := m.tasks[m.cursor].Name
	m.confirming = true
	m.status = fmt.Sprintf(`Really delete %q? [y/N]`, name)
	m.statusErr = false
	return m, nil
}

func (m Model) handleDeleteConfirmed() (tea.Model, tea.Cmd) {
	if len(m.tasks) == 0 {
		return m, nil
	}
	name := m.tasks[m.cursor].Name
	cfg, err := config.Load()
	if err != nil {
		return m, sendError(fmt.Sprintf("load config: %s", err))
	}
	if err := task.Remove(&cfg.Tasks, name); err != nil {
		return m, sendError(fmt.Sprintf("remove: %s", err))
	}
	wrapperDir := filepath.Join(config.ConfigDir(), "wrappers")
	if err := cron.SyncToCrontab(cfg.Tasks, wrapperDir); err != nil {
		return m, sendError(fmt.Sprintf("sync crontab: %s", err))
	}
	if err := config.Save(cfg); err != nil {
		return m, sendError(fmt.Sprintf("save config: %s", err))
	}
	m.tasks = cfg.Tasks
	if m.cursor >= len(m.tasks) && m.cursor > 0 {
		m.cursor = len(m.tasks) - 1
	}
	return m, sendStatus(fmt.Sprintf("Deleted %q", name))
}

func scheduleStatusClear() tea.Cmd {
	return tea.Tick(statusClearDelay, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

func sendStatus(text string) tea.Cmd {
	return func() tea.Msg {
		return statusMsg{text: text, isError: false}
	}
}

func sendError(text string) tea.Cmd {
	return func() tea.Msg {
		return statusMsg{text: text, isError: true}
	}
}

func commandEnv(taskEnvs map[string]string) []string {
	env := os.Environ()
	if len(taskEnvs) == 0 {
		return env
	}
	filtered := make([]string, 0, len(env)+len(taskEnvs))
	for _, entry := range env {
		key, _, ok := splitEnvEntry(entry)
		if ok {
			if _, exists := taskEnvs[key]; exists {
				continue
			}
		}
		filtered = append(filtered, entry)
	}
	for key, value := range taskEnvs {
		filtered = append(filtered, key+"="+value)
	}
	return filtered
}

func splitEnvEntry(entry string) (string, string, bool) {
	for i, c := range entry {
		if c == '=' {
			return entry[:i], entry[i+1:], true
		}
	}
	return entry, "", false
}

func exitCodeFromErr(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}
	return 1
}
