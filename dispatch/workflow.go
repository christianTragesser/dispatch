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
	getClusters(Bucket string) []string
	getClusterCreationDate(Bucket string, cluster string) string
}

func CLICreate(event KopsEvent) KopsEvent {
	createCommand := flag.NewFlagSet("create", flag.ExitOnError)
	createFQDN := createCommand.String("FQDN", "dispatch.k8s.local", "Cluster FQDN")
	createSize := createCommand.String("Size", "small", "cluster node Size")
	nodeCount := createCommand.String("nodes", "2", "cluster node count")
	createVersion := createCommand.String("version", k8sVersion, "Kubernetes version")
	createYOLO := createCommand.Bool("yolo", false, "skip verification prompt for cluster creation")

	err := createCommand.Parse(os.Args[2:])
	if err != nil {
		reportErr(err, " parse create command")
	}

	event.FQDN = *createFQDN
	event.Size = *createSize
	event.Count = *nodeCount
	event.Version = *createVersion
	event.Verified = *createYOLO

	return event
}

func CLIDelete(event KopsEvent) KopsEvent {
	deleteCommand := flag.NewFlagSet("delete", flag.ExitOnError)
	deleteFQDN := deleteCommand.String("FQDN", "", "Cluster FQDN")
	deleteYOLO := deleteCommand.Bool("yolo", false, "skip verification prompt for cluster deletion")

	err := deleteCommand.Parse(os.Args[2:])
	if err != nil {
		reportErr(err, " parse delete command")
	}

	if *deleteFQDN == "" {
		fmt.Print(" ! cluster FQDN is required\n\n")

		return KopsEvent{Action: exitStatus}
	}

	event.FQDN = *deleteFQDN
	event.Verified = *deleteYOLO

	return event
}

func CLIWorkflow(dispatchVersion string, event KopsEvent) KopsEvent {
	action := os.Args[1]

	switch action {
	case "Version":
		fmt.Printf("Dispatch Version %s\n", dispatchVersion)

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
		event.FQDN = createOptions[0]
		event.Size = createOptions[1]
		event.Count = createOptions[2]
		event.Version = k8sVersion

	case deleteAction:
		var clusterList []map[string]string

		existingClusters := te.getClusters(event.Bucket)

		if len(existingClusters) > 0 {
			for _, c := range existingClusters {
				cluster := make(map[string]string)
				cluster["name"] = c
				cluster["date"] = te.getClusterCreationDate(event.Bucket, c)
				clusterList = append(clusterList, cluster)
			}

			event.Action = action
			event.FQDN = te.tuiDelete(clusterList)
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
