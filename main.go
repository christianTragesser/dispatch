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

func main() {
	var sessionEvent dispatch.KopsEvent

	fmt.Print(ASCII_ART)

	if len(os.Args) > 1 {
		sessionEvent = dispatch.CLIOption()
	} else {
		sessionEvent = dispatch.TUIOption()
	}

	sessionEvent = dispatch.EnsureDependencies(sessionEvent)

	dispatch.RunKOPS(sessionEvent)
}
