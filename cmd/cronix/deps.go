package main

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robfig/cron/v3"
)

// Dependency anchors - placeholder references so go mod tidy retains direct deps.
// These will be replaced by real usage in later tasks.
var (
	_ tea.Model  = (*teaPlaceholder)(nil)
	_ list.Item  = (*listPlaceholder)(nil)
	_ lipgloss.Style
	_ *cron.Cron
)

type teaPlaceholder struct{}

func (teaPlaceholder) Init() tea.Cmd                       { return nil }
func (teaPlaceholder) Update(tea.Msg) (tea.Model, tea.Cmd) { return teaPlaceholder{}, nil }
func (teaPlaceholder) View() string                        { return "" }

type listPlaceholder struct{}

func (listPlaceholder) FilterValue() string { return "" }
func (listPlaceholder) Title() string       { return "" }
func (listPlaceholder) Description() string { return "" }
