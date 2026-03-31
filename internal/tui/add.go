package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/QMZCLL/cronix/internal/config"
	"github.com/QMZCLL/cronix/internal/cron"
	"github.com/QMZCLL/cronix/internal/logger"
	"github.com/QMZCLL/cronix/internal/task"
)

const addHelpText = "[tab/shift+tab ↑/↓] navigate  [enter] next/save  [esc] cancel"

const (
	addFieldName = iota
	addFieldCommand
	addFieldDescription
	addFieldMinute
	addFieldHour
	addFieldDayOfMonth
	addFieldMonth
	addFieldDayOfWeek
	addFieldOnce
	addFieldEnvs
	addFieldCount
)

type addState struct {
	inputs []textinput.Model
	focus  int
}

var addStyles = struct {
	container   lipgloss.Style
	header      lipgloss.Style
	label       lipgloss.Style
	activeLabel lipgloss.Style
	helper      lipgloss.Style
	marker      lipgloss.Style
	idleMarker  lipgloss.Style
	section     lipgloss.Style
	note        lipgloss.Style
}{
	container:   tuiStyles.container,
	header:      tuiStyles.pageTitle,
	label:       tuiStyles.fieldLabel,
	activeLabel: tuiStyles.fieldLabelActive,
	helper:      tuiStyles.fieldHelper,
	marker:      tuiStyles.fieldMarker,
	idleMarker:  tuiStyles.fieldMarkerMuted,
	section:     tuiStyles.sectionLabel,
	note:        tuiStyles.inlineNote,
}

func newAddState() addState {
	fields := []struct {
		placeholder string
		value       string
		width       int
	}{
		{placeholder: "backup", width: 32},
		{placeholder: "echo hello", width: 48},
		{placeholder: "optional note", width: 48},
		{placeholder: "0", width: 10},
		{placeholder: "*", width: 10},
		{placeholder: "*", width: 10},
		{placeholder: "*", width: 10},
		{placeholder: "*", width: 10},
		{placeholder: "n", width: 8},
		{placeholder: "KEY=VALUE,KEY2=VALUE2", width: 48},
	}

	inputs := make([]textinput.Model, 0, len(fields))
	for _, field := range fields {
		input := textinput.New()
		input.Prompt = ""
		input.Placeholder = field.placeholder
		input.SetValue(field.value)
		input.Width = field.width
		inputs = append(inputs, input)
	}

	state := addState{inputs: inputs}
	state.setFocus(0)
	return state
}

func (s *addState) setFocus(index int) tea.Cmd {
	if len(s.inputs) == 0 {
		return nil
	}
	if index < 0 {
		index = 0
	}
	if index >= len(s.inputs) {
		index = len(s.inputs) - 1
	}
	for i := range s.inputs {
		s.inputs[i].Blur()
	}
	s.focus = index
	return s.inputs[s.focus].Focus()
}

func (m Model) enterAddPage() (tea.Model, tea.Cmd) {
	m.page = pageAdd
	m.confirming = false
	m.status = ""
	m.statusErr = false
	m.add = newAddState()
	return m, m.add.setFocus(0)
}

func (m Model) updateAdd(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		m.page = pageList
		m.status = ""
		m.statusErr = false
		return m, nil
	case "tab", "down":
		return m, m.add.setFocus(m.add.focus + 1)
	case "shift+tab", "up":
		return m, m.add.setFocus(m.add.focus - 1)
	case "enter":
		if m.add.focus == addFieldCount-1 {
			return m.submitAdd()
		}
		return m, m.add.setFocus(m.add.focus + 1)
	}

	input, cmd := m.add.inputs[m.add.focus].Update(key)
	m.add.inputs[m.add.focus] = input
	return m, cmd
}

func (m Model) submitAdd() (tea.Model, tea.Cmd) {
	minute := inputValueOrDefault(m.add.inputs[addFieldMinute])
	hour := inputValueOrDefault(m.add.inputs[addFieldHour])
	dayOfMonth := inputValueOrDefault(m.add.inputs[addFieldDayOfMonth])
	month := inputValueOrDefault(m.add.inputs[addFieldMonth])
	dayOfWeek := inputValueOrDefault(m.add.inputs[addFieldDayOfWeek])
	cronExpr := strings.Join([]string{minute, hour, dayOfMonth, month, dayOfWeek}, " ")

	runOnce, err := parseOnceValue(inputValueOrDefault(m.add.inputs[addFieldOnce]))
	if err != nil {
		m.status = err.Error()
		m.statusErr = true
		return m, nil
	}

	taskEnvs, err := task.ParseEnvAssignments([]string{m.add.inputs[addFieldEnvs].Value()})
	if err != nil {
		m.status = err.Error()
		m.statusErr = true
		return m, nil
	}

	cfg, err := config.Load()
	if err != nil {
		return m, sendError(fmt.Sprintf("load config: %s", err))
	}

	scheduledTask := task.Task{
		Name:        strings.TrimSpace(m.add.inputs[addFieldName].Value()),
		Command:     strings.TrimSpace(m.add.inputs[addFieldCommand].Value()),
		Description: strings.TrimSpace(m.add.inputs[addFieldDescription].Value()),
		CronExpr:    cronExpr,
		Enabled:     true,
		RunOnce:     runOnce,
		Envs:        taskEnvs,
	}

	if err := task.Add(&cfg.Tasks, scheduledTask); err != nil {
		m.status = err.Error()
		m.statusErr = true
		return m, nil
	}

	logDir := logger.LogDir(cfg.LogDir)
	wrapperDir := filepath.Join(config.ConfigDir(), "wrappers")
	if _, err := cron.WriteWrapper(scheduledTask, wrapperDir, logDir); err != nil {
		return m, sendError(fmt.Sprintf("write wrapper: %s", err))
	}

	cfg.Tasks = normalizeTasks(cfg.Tasks)
	if err := cron.SyncToCrontab(cfg.Tasks, wrapperDir); err != nil {
		return m, sendError(fmt.Sprintf("sync crontab: %s", err))
	}
	if err := config.Save(cfg); err != nil {
		return m, sendError(fmt.Sprintf("save config: %s", err))
	}

	m.page = pageList
	m.tasks = cfg.Tasks
	m.cursor = len(m.tasks) - 1
	m.status = ""
	m.statusErr = false
	return m, sendStatus(fmt.Sprintf("Added %q (%s)", scheduledTask.Name, scheduledTask.CronExpr))
}

func renderAddView(m Model) string {
	cronPreview := strings.Join([]string{
		valueOrPlaceholder(m.add.inputs[addFieldMinute]),
		valueOrPlaceholder(m.add.inputs[addFieldHour]),
		valueOrPlaceholder(m.add.inputs[addFieldDayOfMonth]),
		valueOrPlaceholder(m.add.inputs[addFieldMonth]),
		valueOrPlaceholder(m.add.inputs[addFieldDayOfWeek]),
	}, " ")

	lines := []string{
		renderPageHeading("Add task", "Create a scheduled command. Tab moves focus, enter advances or saves."),
		addStyles.section.Render("Task details"),
		renderAddField(m, addFieldName, "Name", m.add.inputs[addFieldName].View(), "required"),
		renderAddField(m, addFieldCommand, "Command", m.add.inputs[addFieldCommand].View(), "required"),
		renderAddField(m, addFieldDescription, "Description", m.add.inputs[addFieldDescription].View(), "optional"),
		addStyles.section.Render("Schedule"),
		renderAddField(m, addFieldMinute, "Minute", m.add.inputs[addFieldMinute].View(), "cron field 1"),
		renderAddField(m, addFieldHour, "Hour", m.add.inputs[addFieldHour].View(), "cron field 2"),
		renderAddField(m, addFieldDayOfMonth, "Day-of-month", m.add.inputs[addFieldDayOfMonth].View(), "cron field 3"),
		renderAddField(m, addFieldMonth, "Month", m.add.inputs[addFieldMonth].View(), "cron field 4"),
		renderAddField(m, addFieldDayOfWeek, "Day-of-week", m.add.inputs[addFieldDayOfWeek].View(), "cron field 5"),
		lipgloss.NewStyle().BorderLeft(true).BorderForeground(lipgloss.Color("239")).PaddingLeft(1).Render(
			tuiStyles.inlineNoteStrong.Render("Cron preview  ") + addStyles.note.Render(cronPreview),
		),
		addStyles.section.Render("Behavior"),
		renderAddField(m, addFieldOnce, "Once", m.add.inputs[addFieldOnce].View(), "y = disable after first successful scheduled run"),
		renderAddField(m, addFieldEnvs, "Envs", m.add.inputs[addFieldEnvs].View(), "optional comma-separated KEY=VALUE pairs"),
		addStyles.note.Render("Once keeps the cron schedule and only auto-disables after a successful triggered run."),
		renderHelpBlock(addHelpText),
	}

	if m.status != "" {
		lines = append(lines, renderStatusBlock(m.status, m.statusErr, false))
	}

	return addStyles.container.Render(strings.Join(lines, "\n"))
}

func renderAddField(m Model, index int, label, value, helper string) string {
	labelStyle := addStyles.label
	markerStyle := addStyles.idleMarker
	marker := " "
	if m.add.focus == index {
		labelStyle = addStyles.activeLabel
		markerStyle = addStyles.marker
		marker = "›"
	}

	line := fmt.Sprintf("%s %s %s", markerStyle.Render(marker), labelStyle.Render(label), value)
	if helper == "" {
		return line
	}
	return strings.Join([]string{
		line,
		"  " + strings.Repeat(" ", 14) + addStyles.helper.Render(helper),
	}, "\n")
}

func valueOrPlaceholder(input textinput.Model) string {
	value := strings.TrimSpace(input.Value())
	if value == "" {
		return input.Placeholder
	}
	return value
}

func inputValueOrDefault(input textinput.Model) string {
	return strings.TrimSpace(valueOrPlaceholder(input))
}

func parseOnceValue(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "n", "no", "false", "0":
		return false, nil
	case "y", "yes", "true", "1":
		return true, nil
	default:
		return false, fmt.Errorf("invalid once value %q: use y or n", strings.TrimSpace(value))
	}
}
