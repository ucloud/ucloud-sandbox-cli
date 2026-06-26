// Package datetime provides flexible date/time string parsing for CLI flags.
package datetime

import (
	"fmt"
	"time"
)

// timeOnlyLayouts are tried against the current day when the input has no date.
var timeOnlyLayouts = []string{"15:04:05", "15:04"}

// monthDayLayouts use the current year when the input has no year.
var monthDayLayouts = []string{"01-02 15:04:05", "01-02 15:04"}

// fullLayouts include year, month, day, and time.
var fullLayouts = []string{"2006-01-02 15:04:05", "2006-01-02 15:04"}

// Parse parses a flexible datetime string into a local time.Time.
//
// Supported formats (time portion is HH:MM or HH:MM:SS):
//
//	"15:04"              – today at the given time
//	"06-23 15:04"        – this year, June 23 at the given time
//	"2025-07-23 15:04"   – full absolute datetime
func Parse(s string) (time.Time, error) {
	now := time.Now()

	for _, layout := range timeOnlyLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return time.Date(now.Year(), now.Month(), now.Day(),
				t.Hour(), t.Minute(), t.Second(), 0, time.Local), nil
		}
	}

	for _, layout := range monthDayLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return time.Date(now.Year(), t.Month(), t.Day(),
				t.Hour(), t.Minute(), t.Second(), 0, time.Local), nil
		}
	}

	for _, layout := range fullLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported datetime format: %q", s)
}

// ParseSince parses a Go duration string (e.g. "1h", "30m") and returns
// the time that many units before now.
func ParseSince(s string) (time.Time, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid duration %q: %w", s, err)
	}
	return time.Now().Add(-d), nil
}
