package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/QMZCLL/cronix/internal/task"
)

const (
	helpText          = "[a]dd [e]nable [d]isable [r]un [l]ogs [x]delete [q]uit"
	commandColumnSize = 30
	nameColumnSize    = 18
	cronColumnSize    = 17
	statusColumnSize  = 10
	rowMarkerSize     = 2
)

var listStyles = struct {
	container      lipgloss.Style
	header         lipgloss.Style
	selectedMarker lipgloss.Style
	idleMarker     lipgloss.Style
	cell           lipgloss.Style
	cellStrong     lipgloss.Style
	cellMuted      lipgloss.Style
	selectedRow    lipgloss.Style
	enabledStatus  lipgloss.Style
	disabledStatus lipgloss.Style
	onceStatus     lipgloss.Style
	empty          lipgloss.Style
}{
	container:      tuiStyles.container,
	header:         tuiStyles.tableHeader,
	selectedMarker: tuiStyles.selectedMarker,
	idleMarker:     tuiStyles.idleMarker,
	cell:           tuiStyles.tableCell,
	cellStrong:     tuiStyles.tableCellStrong,
	cellMuted:      tuiStyles.tableCellMuted,
	selectedRow:    tuiStyles.selectedRow,
	enabledStatus:  lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true),
	disabledStatus: lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
	onceStatus:     lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true),
	empty:          tuiStyles.emptyState,
}

func renderListView(m Model) string {
	lines := []string{
		renderPageHeading("Cronix tasks", "Inline scheduler overview. Move with ↑/↓ or j/k, then run an action."),
		renderHeader(),
	}
	if len(m.tasks) == 0 {
		lines = append(lines, listStyles.empty.Render("No tasks configured yet.\nPress [a] to add your first task."))
	} else {
		for idx, scheduledTask := range m.tasks {
			lines = append(lines, renderTaskRow(scheduledTask, idx == m.cursor))
		}
	}
	lines = append(lines, renderHelpBlock(helpText))
	if m.status != "" {
		lines = append(lines, renderStatusBlock(m.status, m.statusErr, m.confirming))
	}

	return listStyles.container.Render(strings.Join(lines, "\n"))
}

func renderHeader() string {
	return fmt.Sprintf(
		"%s %s %s %s %s",
		listStyles.idleMarker.Width(rowMarkerSize).Render(""),
		listStyles.header.Width(nameColumnSize).Render("NAME"),
		listStyles.header.Width(cronColumnSize).Render("CRON"),
		listStyles.header.Width(statusColumnSize).Render("STATUS"),
		listStyles.header.Width(commandColumnSize).Render("COMMAND"),
	)
}

func renderTaskRow(scheduledTask task.Task, selected bool) string {
	markerStyle := listStyles.idleMarker
	rowStyle := listStyles.cell
	if selected {
		markerStyle = listStyles.selectedMarker
		rowStyle = rowStyle.Inherit(listStyles.selectedRow)
	}

	marker := markerStyle.Width(rowMarkerSize).Render(markerFor(selected))
	nameStyle := listStyles.cellStrong
	commandStyle := listStyles.cellMuted
	if selected {
		nameStyle = nameStyle.Inherit(listStyles.selectedRow)
		commandStyle = commandStyle.Inherit(listStyles.selectedRow)
	}

	name := nameStyle.Width(nameColumnSize).Render(scheduledTask.Name)
	cronExpr := rowStyle.Width(cronColumnSize).Render(scheduledTask.CronExpr)
	status := renderStatus(scheduledTask)
	command := commandStyle.Width(commandColumnSize).Render(truncateCommand(scheduledTask.Command))

	return fmt.Sprintf("%s %s %s %s %s", marker, name, cronExpr, status, command)
}

func renderStatus(scheduledTask task.Task) string {
	statusStyle := listStyles.disabledStatus
	label := "disabled"
	if scheduledTask.Enabled {
		statusStyle = listStyles.enabledStatus
		if scheduledTask.RunOnce {
			statusStyle = listStyles.onceStatus
			label = "once"
		} else {
			label = "enabled"
		}
	}
	return statusStyle.Width(statusColumnSize).Render(label)
}

func truncateCommand(command string) string {
	runes := []rune(command)
	if len(runes) <= commandColumnSize {
		return command
	}
	return string(runes[:commandColumnSize-3]) + "..."
}

func markerFor(selected bool) string {
	if selected {
		return "›"
	}
	return ""
}
