package task

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
var nowFunc = time.Now

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
	now := nowFunc()
	next := schedule.Next(now)
	if next.IsZero() {
		return "-"
	}
	return formatNextRun(now, next)
}

func formatNextRun(now, next time.Time) string {
	if !next.After(now) {
		return "-"
	}

	now = now.In(next.Location())
	delta := next.Sub(now)

	if delta < time.Hour {
		minutes := int(delta / time.Minute)
		if delta%time.Minute != 0 {
			minutes++
		}
		if minutes < 1 {
			minutes = 1
		}
		return fmt.Sprintf("in %dm", minutes)
	}

	if sameDay(now, next) && delta < 24*time.Hour {
		hours := int(delta / time.Hour)
		minutes := int((delta % time.Hour) / time.Minute)
		if delta%time.Minute != 0 {
			minutes++
			if minutes == 60 {
				hours++
				minutes = 0
			}
		}
		if minutes == 0 {
			return fmt.Sprintf("in %dh", hours)
		}
		return fmt.Sprintf("in %dh %dm", hours, minutes)
	}

	if next.YearDay() == now.YearDay()+1 && next.Year() == now.Year() {
		return next.Format("tomorrow 15:04")
	}

	return next.Format("01-02 15:04")
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
