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
	header    lipgloss.Style
	meta      lipgloss.Style
	help      lipgloss.Style
	empty     lipgloss.Style
	container lipgloss.Style
}{
	header:    tuiStyles.pageTitle,
	meta:      tuiStyles.logsMeta,
	help:      tuiStyles.helpBlock,
	empty:     tuiStyles.emptyState,
	container: tuiStyles.container,
}

const logsHelpText = "[b/esc] back  [p] prev day  [↑/↓] scroll  [pgup/pgdn] page"

func renderLogsView(m Model) string {
	s := m.logs
	headerStr := renderPageHeading(
		"Task logs",
		fmt.Sprintf("Reviewing execution output without leaving inline mode."),
	)
	metaStr := logsStyles.meta.Render(
		fmt.Sprintf("Task  %s    Date  %s", s.taskName, s.date.Format("2006-01-02")),
	)
	vpView := s.viewport.View()
	helpLine := renderHelpBlock(logsHelpText)

	return logsStyles.container.Render(
		strings.Join([]string{headerStr, metaStr, vpView, helpLine}, "\n"),
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
			fmt.Sprintf("No logs for today.\nRun \"cronix run %s\" to generate output.", taskName),
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
