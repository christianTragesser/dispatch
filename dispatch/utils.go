package dispatch

import (
	"flag"
	"fmt"
	"os"
)

func CLIOption() KopsEvent {
	var eventOptions KopsEvent

	action := os.Args[1]

	createCommand := flag.NewFlagSet("create", flag.ExitOnError)
	createName := createCommand.String("name", "dispatch.k8s.local", "cluster name")
	createSize := createCommand.String("size", "small", "cluster node size")
	nodeCount := createCommand.String("nodes", "2", "cluster node count")
	createVersion := createCommand.String("version", "1.21.9", "Kubernetes version")

	deleteCommand := flag.NewFlagSet("delete", flag.ExitOnError)
	deleteName := deleteCommand.String("name", "", "cluster name")

	switch action {
	case "create":
		createCommand.Parse(os.Args[2:])

		eventOptions = KopsEvent{
			action:  action,
			name:    *createName,
			size:    *createSize,
			count:   *nodeCount,
			version: *createVersion,
		}

	case "delete":
		deleteCommand.Parse(os.Args[2:])

		eventOptions = KopsEvent{
			action: action,
			name:   *deleteName,
		}

	case "-h":
		fmt.Printf("Dispatch options:\n dispatch create -h\n dispatch delete -h\n")
		os.Exit(0)

	default:
		fmt.Printf(" ! %s is not a valid Dispatch option\n", os.Args[1])
		fmt.Println("Only available:", createCommand.Args())
		os.Exit(1)
	}

	return eventOptions
}

func EnsureDependencies(kopsSession KopsEvent) KopsEvent {
	fmt.Print("\nEnsuring dependencies:\n")

	ensureInstall("kops")

	kopsSession.user = ensureWorkspace()

	clientConfig := awsClientConfig()

	testAWSCreds(*clientConfig)

	kopsSession.bucket = ensureS3Bucket(*clientConfig, kopsSession.user)

	listClusters(kopsSession.bucket)

	return kopsSession
}

func reportErr(err error, activity string) {
	fmt.Printf(" ! Failed to %s\n\n", activity)
	fmt.Print(err)
	fmt.Print("\n")
	os.Exit(1)
}
