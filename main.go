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

func main() {
	instance := dispatch.Instance{}

	if len(os.Args) > 1 {
		// subcommand provided, use CLI workflow
		session, err := instance.CLIWorkflow()
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		fmt.Print(asciiArt)

		session, err = session.InitInstance()

		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		_, err = session.PulumiExec()
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}
	} else {
		// use TUI workflow
		fmt.Print(asciiArt)

		session, err := instance.InitInstance()
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		session, err = session.TUIWorkflow()
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		_, err = session.PulumiExec()
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}
	}
}
