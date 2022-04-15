package main

import (
	"fmt"
	"os"

	dispatch "github.com/christiantragesser/dispatch/dispatch"
)

var asciiArt = "\n" + `______  _____ _______  _____  _______ _______ _______ _     _
|     \   |   |______ |_____] |_____|    |    |       |_____|
|_____/ __|__ ______| |       |     |    |    |______ |     |   
` + "\n"

var version = "dev-rc"

func main() {
	var sessionEvent dispatch.KopsEvent

	if len(os.Args) > 1 {
		sessionEvent = dispatch.CLIWorkflow(version, sessionEvent)

		if sessionEvent.Action == "exit" {
			os.Exit(0)
		} else {
			fmt.Print(asciiArt)
			sessionEvent = dispatch.EnsureDependencies(sessionEvent)
		}
	} else {
		fmt.Print(asciiArt)

		sessionEvent = dispatch.EnsureDependencies(sessionEvent)

		TUIAPI := dispatch.KopsEvent{}

		sessionEvent = dispatch.TUIWorkflow(TUIAPI, sessionEvent)

		if sessionEvent.Action == "exit" {
			os.Exit(0)
		}
	}

	dispatch.RunKOPS(sessionEvent)
}
