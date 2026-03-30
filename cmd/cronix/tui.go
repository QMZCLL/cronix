package main

import (
	internaltui "github.com/QMZCLL/cronix/internal/tui"
	"github.com/spf13/cobra"
)

func newTUICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Open the interactive task list",
		RunE: func(cmd *cobra.Command, args []string) error {
			return internaltui.StartWithIO(cmd.InOrStdin(), cmd.OutOrStdout())
		},
	}
}
