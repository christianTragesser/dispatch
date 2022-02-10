package main

import (
	"fmt"

	dispatch "github.com/christiantragesser/dispatch/dispatch"
)

var ASCII_ART string = `______  _____ _______  _____  _______ _______ _______ _     _
|     \   |   |______ |_____] |_____|    |    |       |_____|
|_____/ __|__ ______| |       |     |    |    |______ |     |   
` + "\n\n"

func main() {
	fmt.Print(ASCII_ART)
	userID := dispatch.EnsureWorkspace()
	bucket := dispatch.EnsureDependencies(userID)
	dispatch.ListClusters(bucket)
}
