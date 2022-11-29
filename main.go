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

var version = "dev-build"

func main() {
	sessionEvent := &dispatch.Event{}

	if len(os.Args) > 1 {
		// subcommand provided, use CLI workflow
		*sessionEvent = dispatch.CLIWorkflow(version, sessionEvent)

		if sessionEvent.Action == "exit" {
			os.Exit(0)
		} else {
			fmt.Print(asciiArt)
			*sessionEvent = dispatch.EnsureDependencies(sessionEvent)
		}
	} else {
		// use TUI workflow
		fmt.Print(asciiArt)

		*sessionEvent = dispatch.EnsureDependencies(sessionEvent)

		TUIAPI := dispatch.Event{}

		*sessionEvent = dispatch.TUIWorkflow(TUIAPI, sessionEvent)

		if sessionEvent.Action == "exit" {
			os.Exit(0)
		}
	}

	dispatch.Exec(sessionEvent)
}
