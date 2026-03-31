package task

import (
	"time"

	"github.com/robfig/cron/v3"
)

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

func NextRun(t Task) string {
	if !t.Enabled {
		return "-"
	}
	if t.CronExpr == "@reboot" {
		return "@reboot"
	}
	schedule, err := cronParser.Parse(t.CronExpr)
	if err != nil {
		return "-"
	}
	next := schedule.Next(time.Now())
	if next.IsZero() {
		return "-"
	}
	now := time.Now()
	if next.Year() == now.Year() {
		return next.Format("01-02 15:04")
	}
	return next.Format("2006-01-02 15:04")
}
