package main

import (
	"fmt"

	dispatch "github.com/christiantragesser/dispatch/dispatch"
)

var asciiArt = "\n" + `______  _____ _______  _____  _______ _______ _______ _     _
|     \   |   |______ |_____] |_____|    |    |       |_____|
|_____/ __|__ ______| |       |     |    |    |______ |     |   
` + "\n"

var version = "dev-build"

func main() {
	fmt.Print(asciiArt)
	instance := dispatch.Instance{}

	ws, err := instance.SetWorkspace()
	if err != nil {
		fmt.Println(err)
	}

	instance.Home = ws

	err = instance.TestCredentials()
	if err != nil {
		fmt.Println(err)
	}

	instance.Bucket, err = instance.SetBucket()
	if err != nil {
		fmt.Println(err)
	}

	err = instance.ListExistingClusters()
	if err != nil {
		fmt.Println(err)
	}
	/*
		if len(os.Args) > 1 {
			// subcommand provided, use CLI workflow
			sessionEvent = dispatch.CLIWorkflow(version, sessionEvent)

			if sessionEvent.Action == "exit" {
				os.Exit(0)
			}

			fmt.Print(asciiArt)

			sessionEvent = dispatch.EnsureDependencies(sessionEvent)
		} else {
			// use TUI workflow
			fmt.Print(asciiArt)

			sessionEvent = dispatch.EnsureDependencies(sessionEvent)

			TUIAPI := dispatch.Event{}

			sessionEvent = dispatch.TUIWorkflow(TUIAPI, sessionEvent)

			if sessionEvent.Action == "exit" {
				os.Exit(0)
			}
		}

		dispatch.Exec(sessionEvent)
	*/
}
