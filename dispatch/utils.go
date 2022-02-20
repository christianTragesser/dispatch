package dispatch

import (
	"flag"
	"fmt"
	"os"

	"github.com/christiantragesser/dispatch/tuiaction"
	"github.com/christiantragesser/dispatch/tuicreate"
	"github.com/christiantragesser/dispatch/tuidelete"
)

func CLIOption() KopsEvent {
	var eventOptions KopsEvent

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

		eventOptions = KopsEvent{
			action:  action,
			name:    *createName,
			size:    *createSize,
			count:   *nodeCount,
			version: *createVersion,
			verify:  *createYOLO,
		}

	case "delete":
		deleteCommand.Parse(os.Args[2:])

		eventOptions = KopsEvent{
			action: action,
			name:   *deleteName,
			verify: *deleteYOLO,
		}

	case "-h":
		fmt.Printf("Dispatch options:\n dispatch create -h\n dispatch delete -h\n")
		os.Exit(0)

	default:
		fmt.Printf(" ! %s is not a valid Dispatch option\n", action)
		fmt.Printf("\ntry:\n dispatch create -h\n\n or\n\n dispatch delete -h\n\n")
		os.Exit(0)
	}

	return eventOptions
}

func TUIOption() KopsEvent {
	var eventOptions KopsEvent

	action := tuiaction.Action()

	switch action {
	case "create":
		createInfo := tuicreate.Create()

		eventOptions = KopsEvent{
			action:  "create",
			name:    createInfo[0],
			size:    createInfo[1],
			count:   createInfo[2],
			version: "1.21.9",
			verify:  false,
		}

	case "delete":
		deleteInfo := tuidelete.Delete()

		eventOptions = KopsEvent{
			action: "delete",
			name:   deleteInfo[0],
			verify: false,
		}

	default:
		fmt.Printf(" ! %s is not a valid Dispatch option\n", action)
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
