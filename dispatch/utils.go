package dispatch

import (
	"flag"
	"fmt"
	"os"
)

func reportErr(err error, activity string) {
	fmt.Printf(" ! Failed to %s\n\n", activity)
	fmt.Print(err)
	fmt.Print("\n")
	os.Exit(1)
}

func CLIOption(dispatchVersion string, event KopsEvent) KopsEvent {
	action := os.Args[1]

	switch action {
	case "version":
		fmt.Printf("Dispatch version %s\n", dispatchVersion)
		event.Action = "exit"
	case "create":
		createCommand := flag.NewFlagSet("create", flag.ExitOnError)
		createFQDN := createCommand.String("fqdn", "dispatch.k8s.local", "Cluster FQDN")
		createSize := createCommand.String("size", "small", "cluster node size")
		nodeCount := createCommand.String("nodes", "2", "cluster node count")
		createVersion := createCommand.String("version", K8S_VERSION, "Kubernetes version")
		createYOLO := createCommand.Bool("yolo", false, "skip verification prompt for cluster creation")

		createCommand.Parse(os.Args[2:])

		event.Action = action
		event.fqdn = *createFQDN
		event.size = *createSize
		event.count = *nodeCount
		event.version = *createVersion
		event.verified = *createYOLO

	case "delete":
		deleteCommand := flag.NewFlagSet("delete", flag.ExitOnError)
		deleteFQDN := deleteCommand.String("fqdn", "", "Cluster FQDN")
		deleteYOLO := deleteCommand.Bool("yolo", false, "skip verification prompt for cluster deletion")

		deleteCommand.Parse(os.Args[2:])

		if *deleteFQDN == "" {
			fmt.Print(" ! cluster FQDN is required\n\n")

			return KopsEvent{Action: "exit"}
		}

		event.Action = action
		event.fqdn = *deleteFQDN
		event.verified = *deleteYOLO

	case "-h":
		fmt.Printf("Dispatch options:\n dispatch create -h\n dispatch delete -h\n")
		event.Action = "exit"

	default:
		fmt.Printf(" ! %s is not a valid Dispatch option\n", action)
		fmt.Printf("\n dispatch create -h or dispatch delete -h\n")
		event.Action = "exit"
	}

	return event
}

func TUIWorkflow(te TUIEventAPI, event KopsEvent) KopsEvent {
	action := te.getTUIAction()

	switch action {
	case "create":
		createOptions := te.tuiCreate()

		event.Action = action
		event.fqdn = createOptions[0]
		event.size = createOptions[1]
		event.count = createOptions[2]
		event.version = K8S_VERSION

	case "delete":
		var currentClusters []cluster

		clusters := te.getClusters(event.bucket)

		if len(clusters) > 0 {
			for _, c := range clusters {
				item := cluster{}
				item.name = c
				item.date = te.getClusterCreationDate(event.bucket, c)
				currentClusters = append(currentClusters, item)
			}

			event.Action = action
			event.fqdn = te.tuiDelete(currentClusters)

		} else {
			fmt.Print(" . No existing clusters to delete\n")

			return KopsEvent{Action: "exit"}
		}

	default:
		fmt.Printf(" ! %s is not a valid Dispatch option\n", event.Action)

		return KopsEvent{Action: "exit"}
	}

	return event
}

func EnsureDependencies(event KopsEvent) KopsEvent {
	fmt.Print("\nEnsuring dependencies:\n")

	event.user = ensureWorkspace()

	clientConfig := awsClientConfig()

	testAWSCreds(*clientConfig)

	event.bucket = ensureS3Bucket(*clientConfig, event.user)

	listClusters(event.bucket)

	return event
}
