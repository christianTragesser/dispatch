package dispatch

import (
	"fmt"
	"os"
)

func ReportErr(err error, activity string) {
	fmt.Printf(" ! Failed to %s\n\n", activity)
	fmt.Print(err)
	fmt.Print("\n")
	os.Exit(1)
}
