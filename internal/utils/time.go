package utils

import (
	"fmt"
	"github.com/docker/go-units"
	"time"
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
