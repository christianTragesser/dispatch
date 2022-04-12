package main

import (
	"fmt"
	"os"

	dispatch "github.com/christiantragesser/dispatch/dispatch"
)

var ASCII_ART string = "\n" + `______  _____ _______  _____  _______ _______ _______ _     _
|     \   |   |______ |_____] |_____|    |    |       |_____|
|_____/ __|__ ______| |       |     |    |    |______ |     |   
` + "\n"

var version = "dev-rc"

func main() {
	var sessionEvent dispatch.KopsEvent

	if len(os.Args) > 1 {
		sessionEvent = dispatch.CLIOption(version, sessionEvent)

		if sessionEvent.Action == "exit" {
			os.Exit(0)
		} else {
			fmt.Print(ASCII_ART)
			sessionEvent = dispatch.EnsureDependencies(sessionEvent)
		}
	} else {
		fmt.Print(ASCII_ART)

		TUIAPI := dispatch.KopsEvent{}

		sessionEvent = dispatch.EnsureDependencies(sessionEvent)
		sessionEvent = dispatch.TUIWorkflow(TUIAPI, sessionEvent)

		if sessionEvent.Action == "exit" {
			os.Exit(0)
		}
	}

	dispatch.RunKOPS(sessionEvent)
}
