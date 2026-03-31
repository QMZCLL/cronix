package main

import (
	"fmt"
	"io"

	"github.com/QMZCLL/cronix/internal/cron"
)

func writeCronDaemonWarning(output io.Writer) {
	warning := cron.CronDaemonWarning()
	if warning == "" {
		return
	}
	_, _ = fmt.Fprintf(output, "! %s\n", warning)
}
