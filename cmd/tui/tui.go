package main

import (
	"log"

	"github.com/signadot/cli/internal/tui"
)

func main() {
	trafficWatch := tui.NewTrafficWatch("testdata/traffic")
	if err := trafficWatch.Run(); err != nil {
		log.Fatalf("error running traffic watch: %v", err)
	}
}
