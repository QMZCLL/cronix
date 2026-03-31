package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const verticalPadding = 1

var tuiStyles = struct {
	container        lipgloss.Style
	pageTitle        lipgloss.Style
	pageSubtitle     lipgloss.Style
	sectionLabel     lipgloss.Style
	helpBlock        lipgloss.Style
	statusBase       lipgloss.Style
	statusOk         lipgloss.Style
	statusErr        lipgloss.Style
	statusConfirm    lipgloss.Style
	emptyState       lipgloss.Style
	tableHeader      lipgloss.Style
	tableCell        lipgloss.Style
	tableCellStrong  lipgloss.Style
	tableCellMuted   lipgloss.Style
	selectedRow      lipgloss.Style
	selectedMarker   lipgloss.Style
	idleMarker       lipgloss.Style
	fieldMarker      lipgloss.Style
	fieldMarkerMuted lipgloss.Style
	fieldLabel       lipgloss.Style
	fieldLabelActive lipgloss.Style
	fieldHelper      lipgloss.Style
	inlineNote       lipgloss.Style
	inlineNoteStrong lipgloss.Style
	logsMeta         lipgloss.Style
}{
	container:        lipgloss.NewStyle().Padding(verticalPadding, 2),
	pageTitle:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")),
	pageSubtitle:     lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	sectionLabel:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).MarginTop(1),
	helpBlock:        lipgloss.NewStyle().Foreground(lipgloss.Color("245")).BorderLeft(true).BorderForeground(lipgloss.Color("239")).PaddingLeft(1).MarginTop(1),
	statusBase:       lipgloss.NewStyle().Bold(true).BorderLeft(true).PaddingLeft(1).MarginTop(1),
	statusOk:         lipgloss.NewStyle().Foreground(lipgloss.Color("10")).BorderForeground(lipgloss.Color("10")),
	statusErr:        lipgloss.NewStyle().Foreground(lipgloss.Color("9")).BorderForeground(lipgloss.Color("9")),
	statusConfirm:    lipgloss.NewStyle().Foreground(lipgloss.Color("11")).BorderForeground(lipgloss.Color("11")),
	emptyState:       lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("239")).Padding(0, 1).MarginTop(1),
	tableHeader:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("250")).BorderBottom(true).BorderForeground(lipgloss.Color("239")),
	tableCell:        lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
	tableCellStrong:  lipgloss.NewStyle().Foreground(lipgloss.Color("15")),
	tableCellMuted:   lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	selectedRow:      lipgloss.NewStyle().Bold(true).Underline(true),
	selectedMarker:   lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true),
	idleMarker:       lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
	fieldMarker:      lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true),
	fieldMarkerMuted: lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
	fieldLabel:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")).Width(14),
	fieldLabelActive: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Width(14),
	fieldHelper:      lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	inlineNote:       lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	inlineNoteStrong: lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true),
	logsMeta:         lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true),
}

func renderPageHeading(title, subtitle string) string {
	if subtitle == "" {
		return tuiStyles.pageTitle.Render(title)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		tuiStyles.pageTitle.Render(title),
		tuiStyles.pageSubtitle.Render(subtitle),
	)
}

func renderPageHeadingWithClock(title, subtitle, clock string, width int) string {
	titleLine := tuiStyles.pageTitle.Render(title)
	if clock != "" {
		clockStyle := tuiStyles.pageSubtitle.Bold(true)
		if width > 0 {
			contentWidth := max(width-4, 0)
			titleLine = lipgloss.JoinHorizontal(
				lipgloss.Top,
				lipgloss.NewStyle().Width(max(contentWidth-lipgloss.Width(clockStyle.Render(clock))-1, 0)).Render(titleLine),
				clockStyle.Render(clock),
			)
		} else {
			titleLine = lipgloss.JoinHorizontal(lipgloss.Top, titleLine, " ", clockStyle.Render(clock))
		}
	}

	if subtitle == "" {
		return titleLine
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleLine,
		tuiStyles.pageSubtitle.Render(subtitle),
	)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func renderHelpBlock(text string) string {
	return tuiStyles.helpBlock.Render("Keys  " + text)
}

func renderStatusBlock(text string, isError bool, confirming bool) string {
	style := tuiStyles.statusBase.Inherit(tuiStyles.statusOk)
	label := "STATUS"
	if isError {
		style = tuiStyles.statusBase.Inherit(tuiStyles.statusErr)
		label = "ERROR"
	} else if confirming {
		style = tuiStyles.statusBase.Inherit(tuiStyles.statusConfirm)
		label = "CONFIRM"
	}

	return style.Render(fmt.Sprintf("%s  %s", label, text))
}

func renderEmptyState(lines ...string) string {
	return tuiStyles.emptyState.Render(strings.Join(lines, "\n"))
}
