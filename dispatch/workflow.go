package dispatch

import (
	"flag"
	"fmt"
	"os"
)

type TUIEventAPI interface {
	getTUIAction() string
	tuiCreate() []string
	tuiDelete(cluster []map[string]string) string
	getClusters(bucket string) []string
	getClusterCreationDate(bucket string, cluster string) string
}

func CLICreate(event KopsEvent) KopsEvent {
	createCommand := flag.NewFlagSet("create", flag.ExitOnError)
	createFQDN := createCommand.String("fqdn", "dispatch.k8s.local", "Cluster FQDN")
	createSize := createCommand.String("size", "small", "cluster node size")
	nodeCount := createCommand.String("nodes", "2", "cluster node count")
	createVersion := createCommand.String("version", k8sVersion, "Kubernetes version")
	createYOLO := createCommand.Bool("yolo", false, "skip verification prompt for cluster creation")

	err := createCommand.Parse(os.Args[2:])
	if err != nil {
		reportErr(err, " parse create command")
	}

	event.fqdn = *createFQDN
	event.size = *createSize
	event.count = *nodeCount
	event.version = *createVersion
	event.verified = *createYOLO

	return event
}

func CLIDelete(event KopsEvent) KopsEvent {
	deleteCommand := flag.NewFlagSet("delete", flag.ExitOnError)
	deleteFQDN := deleteCommand.String("fqdn", "", "Cluster FQDN")
	deleteYOLO := deleteCommand.Bool("yolo", false, "skip verification prompt for cluster deletion")

	err := deleteCommand.Parse(os.Args[2:])
	if err != nil {
		reportErr(err, " parse delete command")
	}

	if *deleteFQDN == "" {
		fmt.Print(" ! cluster FQDN is required\n\n")

		return KopsEvent{Action: exitStatus}
	}

	event.fqdn = *deleteFQDN
	event.verified = *deleteYOLO

	return event
}

func CLIWorkflow(dispatchVersion string, event KopsEvent) KopsEvent {
	action := os.Args[1]

	switch action {
	case "version":
		fmt.Printf("Dispatch version %s\n", dispatchVersion)

		event.Action = exitStatus
	case "create":
		event = CLICreate(event)
		event.Action = action

	case "delete":
		event = CLIDelete(event)
		event.Action = action

	case "-h":
		fmt.Printf("Dispatch options:\n dispatch create -h\n dispatch delete -h\n")

		event.Action = exitStatus

	default:
		fmt.Printf(" ! %s is not a valid Dispatch option\n", action)
		fmt.Printf("\n dispatch create -h or dispatch delete -h\n")

		event.Action = exitStatus
	}

	return event
}

func TUIWorkflow(te TUIEventAPI, event KopsEvent) KopsEvent {
	action := te.getTUIAction()

	switch action {
	case createAction:
		createOptions := te.tuiCreate()

		event.Action = action
		event.fqdn = createOptions[0]
		event.size = createOptions[1]
		event.count = createOptions[2]
		event.version = k8sVersion

	case deleteAction:
		var clusterList []map[string]string

		existingClusters := te.getClusters(event.bucket)

		if len(existingClusters) > 0 {
			for _, c := range existingClusters {
				cluster := make(map[string]string)
				cluster["name"] = c
				cluster["date"] = te.getClusterCreationDate(event.bucket, c)
				clusterList = append(clusterList, cluster)
			}

			event.Action = action
			event.fqdn = te.tuiDelete(clusterList)
		} else {
			fmt.Print(" . No existing clusters to delete\n")

			return KopsEvent{Action: exitStatus}
		}

	default:
		fmt.Printf(" ! %s is not a valid Dispatch option\n", event.Action)

		return KopsEvent{Action: exitStatus}
	}

	return event
}
