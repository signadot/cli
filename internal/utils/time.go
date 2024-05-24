package utils

import (
	"fmt"
	"strconv"
	"time"

	"github.com/docker/go-units"
	"github.com/xeonx/timeago"
)

func FormatTimestamp(in string) string {
	t, err := time.Parse(time.RFC3339, in)
	if err != nil {
		return in
	}
	elapsed := units.HumanDuration(time.Since(t))
	local := t.Local().Format(time.RFC1123)

	return fmt.Sprintf("%s (%s ago)", local, elapsed)
}

func GetTTLTimeAgoFromBase(baseTime time.Time, dur string) string {
	n := len(dur)
	count, unit := dur[0:n-1], dur[n-1:]
	m, err := strconv.ParseInt(count, 10, 64)
	if err != nil {
		return "?(e parse dur)"
	}
	if m < 0 {
		return "?(e negative dur)"
	}

	offset := time.Duration(m)
	switch unit {
	case "m":
		offset *= time.Minute
	case "h":
		offset *= time.Hour
	case "d":
		offset *= 24 * time.Hour
	case "w":
		offset *= 24 * 7 * time.Hour
	}

	eol := baseTime.Add(offset)
	return timeago.NoMax(timeago.English).Format(eol)
}
