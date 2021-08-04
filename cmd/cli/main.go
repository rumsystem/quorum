package main

import (
	"github.com/huo-ju/quorum/cmd/cli/config"
	"github.com/huo-ju/quorum/cmd/cli/ui"
)

func main() {
	config.Init()
	ui.Init()

	if err := ui.App.Run(); err != nil {
		panic(err)
	}
}
