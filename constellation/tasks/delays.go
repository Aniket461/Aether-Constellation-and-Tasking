package tasks

import (
	"fmt"
	"strings"
	"time"
)

// Delays is how long a task stays in downlink, then processing, after imaging ends.
// Zero skips that phase. The imaging interval itself does not grow.
type Delays struct {
	Downlink   time.Duration
	Processing time.Duration
}

// ParseDelays reads TASK_DOWNLINK_DELAY and TASK_PROCESSING_DELAY.
// An empty value uses 15s and 20s. A value that is not a Go duration, or is negative, is an error.
func ParseDelays(getenv func(string) string) (Delays, error) {
	downlink, err := parseDelay(getenv, "TASK_DOWNLINK_DELAY", 15*time.Second)
	if err != nil {
		return Delays{}, err
	}
	processing, err := parseDelay(getenv, "TASK_PROCESSING_DELAY", 20*time.Second)
	if err != nil {
		return Delays{}, err
	}
	return Delays{Downlink: downlink, Processing: processing}, nil
}

func parseDelay(getenv func(string) string, name string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(getenv(name))
	if raw == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	if d < 0 {
		return 0, fmt.Errorf("%s must not be negative", name)
	}
	return d, nil
}

func (d Delays) clamp() Delays {
	if d.Downlink < 0 {
		d.Downlink = 0
	}
	if d.Processing < 0 {
		d.Processing = 0
	}
	return d
}
