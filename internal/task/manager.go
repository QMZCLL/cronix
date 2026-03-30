package task

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func Add(tasks *[]Task, scheduledTask Task) error {
	if tasks == nil {
		return fmt.Errorf("task: tasks is nil")
	}
	if scheduledTask.Name == "" {
		return fmt.Errorf("task: task name is required")
	}
	if _, err := FindByName(*tasks, scheduledTask.Name); err == nil {
		return fmt.Errorf("task %q already exists", scheduledTask.Name)
	}
	if err := ValidateCronExpr(scheduledTask.CronExpr); err != nil {
		return err
	}
	if scheduledTask.CreatedAt.IsZero() {
		scheduledTask.CreatedAt = time.Now().UTC()
	}
	*tasks = append(*tasks, scheduledTask)
	return nil
}

func Remove(tasks *[]Task, name string) error {
	if tasks == nil {
		return fmt.Errorf("task: tasks is nil")
	}
	idx := indexByName(*tasks, name)
	if idx == -1 {
		return fmt.Errorf("task %q not found", name)
	}
	*tasks = append((*tasks)[:idx], (*tasks)[idx+1:]...)
	return nil
}

func Enable(tasks []Task, name string) error {
	scheduledTask, err := FindByName(tasks, name)
	if err != nil {
		return err
	}
	scheduledTask.Enabled = true
	return nil
}

func Disable(tasks []Task, name string) error {
	scheduledTask, err := FindByName(tasks, name)
	if err != nil {
		return err
	}
	scheduledTask.Enabled = false
	return nil
}

func List(tasks []Task) []Task {
	return tasks
}

func FindByName(tasks []Task, name string) (*Task, error) {
	idx := indexByName(tasks, name)
	if idx == -1 {
		return nil, fmt.Errorf("task %q not found", name)
	}
	return &tasks[idx], nil
}

func ValidateCronExpr(expr string) error {
	if expr == "@reboot" {
		return nil
	}
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return fmt.Errorf("invalid cron expression: expected 5 fields")
	}

	ranges := [][2]int{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 7}}
	for idx, field := range fields {
		if err := validateCronField(field, ranges[idx][0], ranges[idx][1]); err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
	}

	return nil
}

func indexByName(tasks []Task, name string) int {
	for idx := range tasks {
		if tasks[idx].Name == name {
			return idx
		}
	}
	return -1
}

func validateCronField(field string, min, max int) error {
	parts := strings.Split(field, ",")
	for _, part := range parts {
		if err := validateCronPart(part, min, max); err != nil {
			return err
		}
	}
	return nil
}

func validateCronPart(part string, min, max int) error {
	if part == "" {
		return fmt.Errorf("empty field")
	}

	base := part
	if strings.Contains(part, "/") {
		segments := strings.Split(part, "/")
		if len(segments) != 2 {
			return fmt.Errorf("invalid step %q", part)
		}
		base = segments[0]
		step, err := strconv.Atoi(segments[1])
		if err != nil || step <= 0 {
			return fmt.Errorf("invalid step %q", part)
		}
	}

	if base == "*" {
		return nil
	}

	if strings.Contains(base, "-") {
		bounds := strings.Split(base, "-")
		if len(bounds) != 2 {
			return fmt.Errorf("invalid range %q", part)
		}
		start, err := strconv.Atoi(bounds[0])
		if err != nil {
			return fmt.Errorf("invalid range %q", part)
		}
		end, err := strconv.Atoi(bounds[1])
		if err != nil {
			return fmt.Errorf("invalid range %q", part)
		}
		if start < min || end > max || start > end {
			return fmt.Errorf("field %q out of range", part)
		}
		return nil
	}

	value, err := strconv.Atoi(base)
	if err != nil {
		return fmt.Errorf("invalid value %q", part)
	}
	if value < min || value > max {
		return fmt.Errorf("field %q out of range", part)
	}
	return nil
}
