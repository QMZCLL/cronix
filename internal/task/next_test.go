package task

import (
	"testing"
	"time"
)

func TestNextRun(t *testing.T) {
	originalNowFunc := nowFunc
	t.Cleanup(func() {
		nowFunc = originalNowFunc
	})

	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, 3, 31, 17, 20, 0, 0, loc)
	nowFunc = func() time.Time {
		return now
	}

	tests := []struct {
		name string
		task Task
		want string
	}{
		{
			name: "disabled task returns dash",
			task: Task{CronExpr: "*/5 * * * *", Enabled: false},
			want: "-",
		},
		{
			name: "reboot task stays reboot",
			task: Task{CronExpr: "@reboot", Enabled: true},
			want: "@reboot",
		},
		{
			name: "sub hour renders minutes",
			task: Task{CronExpr: "38 17 * * *", Enabled: true},
			want: "in 18m",
		},
		{
			name: "same day renders hours and minutes",
			task: Task{CronExpr: "5 20 * * *", Enabled: true},
			want: "in 2h 45m",
		},
		{
			name: "next day renders tomorrow",
			task: Task{CronExpr: "38 17 * * *", Enabled: true},
			want: "tomorrow 17:38",
		},
		{
			name: "later date renders month and day",
			task: Task{CronExpr: "0 9 4 4 *", Enabled: true},
			want: "04-04 09:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NextRun(tt.task)
			if tt.name == "next day renders tomorrow" {
				nowFunc = func() time.Time {
					return time.Date(2026, 3, 31, 17, 39, 0, 0, loc)
				}
				defer func() {
					nowFunc = func() time.Time {
						return now
					}
				}()
				got = NextRun(tt.task)
			}
			if got != tt.want {
				t.Fatalf("NextRun() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatNextRun_RoundsUpPredictably(t *testing.T) {
	loc := time.FixedZone("UTC", 0)
	now := time.Date(2026, 3, 31, 10, 0, 1, 0, loc)

	tests := []struct {
		name string
		next time.Time
		want string
	}{
		{
			name: "minutes round up",
			next: time.Date(2026, 3, 31, 10, 18, 0, 0, loc),
			want: "in 18m",
		},
		{
			name: "same day hour boundary rounds up",
			next: time.Date(2026, 3, 31, 12, 0, 0, 0, loc),
			want: "in 2h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatNextRun(now, tt.next)
			if got != tt.want {
				t.Fatalf("formatNextRun() = %q, want %q", got, tt.want)
			}
		})
	}
}
