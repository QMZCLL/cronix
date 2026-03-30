package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	"github.com/QMZCLL/cronix/internal/logger"
)

var logsStyles = struct {
	header  lipgloss.Style
	help    lipgloss.Style
	empty   lipgloss.Style
	container lipgloss.Style
}{
	header:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")),
	help:      lipgloss.NewStyle().Foreground(lipgloss.Color("245")).MarginTop(1),
	empty:     lipgloss.NewStyle().Foreground(lipgloss.Color("245")).MarginTop(1),
	container: lipgloss.NewStyle().Padding(verticalPadding, 2),
}

const logsHelpText = "[b/esc] back  [p] prev day  [↑/↓] scroll  [pgup/pgdn] page"

func renderLogsView(m Model) string {
	s := m.logs
	headerStr := logsStyles.header.Render(
		fmt.Sprintf("Logs: %s | %s", s.taskName, s.date.Format("2006-01-02")),
	)
	vpView := s.viewport.View()
	helpLine := logsStyles.help.Render(logsHelpText)

	return logsStyles.container.Render(
		strings.Join([]string{headerStr, vpView, helpLine}, "\n"),
	)
}

func loadLogContent(taskName string, date time.Time) string {
	logDir := logger.LogDir("")
	path := logger.LogPath(logDir, taskName, date)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func buildLogsState(taskName string, date time.Time, width, height int) logsState {
	content := loadLogContent(taskName, date)

	vpHeight := height - 4
	if vpHeight < 1 {
		vpHeight = 1
	}
	vp := viewport.New(width, vpHeight)

	if content == "" {
		vp.SetContent(logsStyles.empty.Render(
			fmt.Sprintf("No logs for today. Run \"cronix run %s\" to generate.", taskName),
		))
	} else {
		vp.SetContent(content)
	}

	return logsState{
		taskName: taskName,
		date:     date,
		viewport: vp,
		content:  content,
	}
}
