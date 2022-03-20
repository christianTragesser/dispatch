package main

import (
	"fmt"
	"os"

	dispatch "github.com/christiantragesser/dispatch/dispatch"
)

var version string
var ASCII_ART string = "\n" + `______  _____ _______  _____  _______ _______ _______ _     _
|     \   |   |______ |_____] |_____|    |    |       |_____|
|_____/ __|__ ______| |       |     |    |    |______ |     |   
` + "\n"

func main() {
	var sessionEvent dispatch.KopsEvent

	if len(os.Args) > 1 {
		if os.Args[1] == "version" {
			fmt.Printf("Dispatch %s\n\n", version)
			os.Exit(0)
		} else {
			dispatch.CLIOption(sessionEvent)
		}
	}

	fmt.Print(ASCII_ART)

	sessionEvent = dispatch.EnsureDependencies()

	if len(os.Args) > 1 {
		sessionEvent = dispatch.CLIOption(sessionEvent)
	} else {
		sessionEvent = dispatch.TUIOption(sessionEvent)
	}

	dispatch.RunKOPS(sessionEvent)
}
