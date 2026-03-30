package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/QMZCLL/cronix/internal/task"
)

const (
	helpText          = "[e]nable [d]isable [r]un [l]ogs [x]delete [q]uit"
	commandColumnSize = 30
	nameColumnSize    = 18
	cronColumnSize    = 17
	statusColumnSize  = 10
	rowMarkerSize     = 2
	verticalPadding   = 1
)

var listStyles = struct {
	container      lipgloss.Style
	header         lipgloss.Style
	selectedMarker lipgloss.Style
	idleMarker     lipgloss.Style
	cell           lipgloss.Style
	selectedRow    lipgloss.Style
	enabledStatus  lipgloss.Style
	disabledStatus lipgloss.Style
	help           lipgloss.Style
	empty          lipgloss.Style
	statusOk       lipgloss.Style
	statusErr      lipgloss.Style
}{
	container:      lipgloss.NewStyle().Padding(verticalPadding, 2),
	header:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")),
	selectedMarker: lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true),
	idleMarker:     lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
	cell:           lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
	selectedRow:    lipgloss.NewStyle().Bold(true),
	enabledStatus:  lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true),
	disabledStatus: lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
	help:           lipgloss.NewStyle().Foreground(lipgloss.Color("245")).MarginTop(1),
	empty:          lipgloss.NewStyle().Foreground(lipgloss.Color("245")).MarginTop(1),
	statusOk:       lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
	statusErr:      lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
}

func renderListView(m Model) string {
	lines := []string{renderHeader()}
	if len(m.tasks) == 0 {
		lines = append(lines, listStyles.empty.Render("No tasks configured yet."))
	} else {
		for idx, scheduledTask := range m.tasks {
			lines = append(lines, renderTaskRow(scheduledTask, idx == m.cursor))
		}
	}
	lines = append(lines, listStyles.help.Render(helpText))
	if m.status != "" {
		if m.statusErr {
			lines = append(lines, listStyles.statusErr.Render(m.status))
		} else {
			lines = append(lines, listStyles.statusOk.Render(m.status))
		}
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
	name := rowStyle.Width(nameColumnSize).Render(scheduledTask.Name)
	cronExpr := rowStyle.Width(cronColumnSize).Render(scheduledTask.CronExpr)
	status := renderStatus(scheduledTask.Enabled)
	command := rowStyle.Width(commandColumnSize).Render(truncateCommand(scheduledTask.Command))

	return fmt.Sprintf("%s %s %s %s %s", marker, name, cronExpr, status, command)
}

func renderStatus(enabled bool) string {
	statusStyle := listStyles.disabledStatus
	label := "disabled"
	if enabled {
		statusStyle = listStyles.enabledStatus
		label = "enabled"
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
