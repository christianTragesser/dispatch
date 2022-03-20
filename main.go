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
	}

	fmt.Print(ASCII_ART)

	sessionEvent = dispatch.EnsureDependencies()

	if len(os.Args) == 0 {
		sessionEvent = dispatch.TUIOption(sessionEvent)
	}

	dispatch.RunKOPS(sessionEvent)
}
