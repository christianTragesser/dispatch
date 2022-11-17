package dispatch

import (
	"fmt"
	"os"
	"strings"
)

func reportErr(err error, activity string) {
	fmt.Printf(" ! Failed to %s\n\n", activity)
	fmt.Print(err)
	fmt.Print("\n")
	os.Exit(1)
}

func clusterExists(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func EnsureDependencies(event DispatchEvent) DispatchEvent {
	fmt.Print("\nEnsuring dependencies:\n")

	event.User = ensureWorkspace()

	clientConfig := awsClientConfig()

	testAWSCreds(*clientConfig)

	event.Bucket = ensureS3Bucket(*clientConfig, event.User)

	printExistingClusters(event.Bucket)

	return event
}

func getNodeSize(size string) (string, error) {
	var ec2Instance string

	nodeSize := strings.ToUpper(size)

	switch nodeSize {
	case "SMALL", "S":
		ec2Instance = smallEC2
	case "MEDIUM", "M":
		ec2Instance = mediumEC2
	case "LARGE", "L":
		ec2Instance = largeEC2
	default:
		return "", fmt.Errorf("invalid node size: %s", size)
	}

	return ec2Instance, nil
}
