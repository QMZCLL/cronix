package tui

import (
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/QMZCLL/cronix/internal/config"
)

func Start() error {
	return StartWithIO(os.Stdin, os.Stdout)
}

func StartWithIO(input io.Reader, output io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	program := tea.NewProgram(
		NewModel(cfg.Tasks),
		tea.WithInput(input),
		tea.WithOutput(output),
	)

	_, err = program.Run()
	return err
}
