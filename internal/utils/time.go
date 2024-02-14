package utils

import (
	"bytes"
	"fmt"
	"math"
	"time"
)

const (
	minute  = 1
	hour    = minute * 60
	day     = hour * 24
	month   = day * 30
	year    = day * 365
	quarter = year / 4
)

// FromDurationGetTimeAgo returns a friendly string representing an approximation of the
// given duration
func FromDurationGetTimeAgo(d time.Duration) string {
	seconds := round(d.Seconds())

	if seconds < 30 {
		return "less than a minute"
	}

	if seconds < 90 {
		return "1 minute"
	}

	minutes := div(seconds, 60)

	if minutes < 45 {
		return fmt.Sprintf("%0d minutes", minutes)
	}

	hours := div(minutes, 60)

	if minutes < day {
		return fmt.Sprintf("%s", pluralize(hours, "hour"))
	}

	if minutes < (42 * hour) {
		return "1 day"
	}

	days := div(hours, 24)

	if minutes < (30 * day) {
		return pluralize(days, "day")
	}

	months := div(days, 30)

	if minutes < (45 * day) {
		return "1 month"
	}

	if minutes < (60 * day) {
		return "2 months"
	}

	if minutes < year {
		return pluralize(months, "month")
	}

	years := minutes / year
	return fmt.Sprintf("%s", pluralize(years, "year"))
}

// FromTimeGetStringAgo returns a friendly string representing the approximate difference
// from the given time and time.Now()
func FromTimeGetStringAgo(t time.Time) string {
	now := time.Now()

	var d time.Duration
	var suffix string

	if t.Before(now) {
		d = now.Sub(t)
		suffix = "ago"
	} else {
		d = t.Sub(now)
		suffix = "from now"
	}

	return fmt.Sprintf("%s %s", FromDurationGetTimeAgo(d), suffix)
}

func pluralize(i int, s string) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%d %s", i, s))
	if i != 1 {
		buf.WriteString("s")
	}
	return buf.String()
}

func round(f float64) int {
	return int(math.Floor(f + .50))
}

func div(numerator int, denominator int) int {
	rem := numerator % denominator
	result := numerator / denominator

	if rem >= (denominator / 2) {
		result++
	}

	return result
}
