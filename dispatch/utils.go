package dispatch

import (
	"flag"
	"fmt"
	"os"

	"github.com/christiantragesser/dispatch/tuiaction"
	"github.com/christiantragesser/dispatch/tuicreate"
	"github.com/christiantragesser/dispatch/tuidelete"
)

func CLIOption(event KopsEvent) KopsEvent {
	action := os.Args[1]

	createCommand := flag.NewFlagSet("create", flag.ExitOnError)
	createName := createCommand.String("name", "dispatch.k8s.local", "cluster name")
	createSize := createCommand.String("size", "small", "cluster node size")
	nodeCount := createCommand.String("nodes", "2", "cluster node count")
	createVersion := createCommand.String("version", "1.21.9", "Kubernetes version")
	createYOLO := createCommand.Bool("yolo", false, "skip verification prompt for cluster creation")

	deleteCommand := flag.NewFlagSet("delete", flag.ExitOnError)
	deleteName := deleteCommand.String("name", "", "cluster name")
	deleteYOLO := deleteCommand.Bool("yolo", false, "skip verification prompt for cluster deletion")

	switch action {
	case "create":
		createCommand.Parse(os.Args[2:])

		event.action = action
		event.name = *createName
		event.size = *createSize
		event.count = *nodeCount
		event.version = *createVersion
		event.verify = *createYOLO

	case "delete":
		deleteCommand.Parse(os.Args[2:])

		event.action = action
		event.name = *deleteName
		event.verify = *deleteYOLO

	case "-h":
		fmt.Printf("Dispatch options:\n dispatch create -h\n dispatch delete -h\n")
		os.Exit(0)

	default:
		fmt.Printf(" ! %s is not a valid Dispatch option\n", action)
		fmt.Printf("\ntry:\n dispatch create -h\n\n or\n\n dispatch delete -h\n\n")
		os.Exit(0)
	}

	return event
}

func TUIOption(event KopsEvent) KopsEvent {
	action := tuiaction.Action()

	switch action {
	case "create":
		createInfo := tuicreate.Create()

		event.action = action
		event.name = createInfo[0]
		event.size = createInfo[1]
		event.count = createInfo[2]
		event.version = "1.21.9"

	case "delete":
		deleteInfo := tuidelete.Delete()

		event.action = action
		event.name = deleteInfo[0]

	default:
		fmt.Printf(" ! %s is not a valid Dispatch option\n", action)
		os.Exit(1)
	}

	return event
}

func EnsureDependencies() KopsEvent {
	var kopsSession KopsEvent

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
