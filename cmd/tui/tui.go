package main

import (
	"log"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/tui"
)

func main() {
	trafficWatch := tui.NewTrafficWatch("/home/davixcky/.signadot/traffic/watch-json", config.OutputFormatYAML)
	if err := trafficWatch.Run(); err != nil {
		log.Fatalf("error running traffic watch: %v", err)
	}
}
