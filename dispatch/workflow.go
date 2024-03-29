package dispatch

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type TUIEventAPI interface {
	getTUIAction() string
	tuiCreate() []string
	tuiDelete(cluster []map[string]string) string
	getClusters(Bucket string) []string
	getClusterCreationDate(Bucket string, cluster string) string
}

func CLICreate(event *Event) Event {
	createCommand := flag.NewFlagSet("create", flag.ExitOnError)
	createName := createCommand.String("name", "", "cluster name")
	createSize := createCommand.String("size", "small", "cluster node size")
	nodeCount := createCommand.String("nodes", "2", "cluster node count")
	createVersion := createCommand.String("version", k8sVersion, "Kubernetes version")
	createYOLO := createCommand.Bool("yes", false, "skip verification prompt for cluster creation")

	err := createCommand.Parse(os.Args[2:])
	if err != nil {
		reportErr(err, " parse create command")
	}

	event.Name = strings.ToLower(*createName)
	event.Size = *createSize
	event.Count = *nodeCount
	event.Version = *createVersion
	event.Verified = *createYOLO

	return *event
}

func CLIDelete(event *Event) Event {
	deleteCommand := flag.NewFlagSet("delete", flag.ExitOnError)
	deleteName := deleteCommand.String("name", "", "cluster name")
	deleteYOLO := deleteCommand.Bool("yes", false, "skip verification prompt for cluster deletion")

	err := deleteCommand.Parse(os.Args[2:])
	if err != nil {
		reportErr(err, " parse delete command")
	}

	event.Name = strings.ToLower(*deleteName)
	event.Verified = *deleteYOLO

	return *event
}

func CLIWorkflow(dispatchVersion string, event *Event) Event {
	action := os.Args[1]

	switch action {
	case "version", "-v":
		fmt.Printf("Dispatch Version %s\n", dispatchVersion)

		event.Action = exitStatus
	case "create":
		*event = CLICreate(event)
		event.Action = action

		if event.Name == "" {
			fmt.Println(" ! create events require the -name flag")

			event.Action = exitStatus
		} else {
			_, err := validateClusterName(event.Name)
			if err != nil {
				reportErr(err, "provide valid cluster name")
			}
		}

	case "delete":
		*event = CLIDelete(event)
		event.Action = action

		if event.Name == "" {
			fmt.Println(" ! delete events require the -name flag")

			event.Action = exitStatus
		} else {
			_, err := validateClusterName(event.Name)
			if err != nil {
				reportErr(err, "provide valid cluster name")
			}
		}

	case "-h":
		fmt.Printf("Dispatch options:\n dispatch create -h\n dispatch delete -h\n")

		event.Action = exitStatus

	default:
		fmt.Printf(" ! %s is not a valid Dispatch option\n", action)
		fmt.Printf("\n dispatch create -h or dispatch delete -h\n")

		event.Action = exitStatus
	}

	return *event
}

func TUIWorkflow(te TUIEventAPI, event *Event) Event {
	action := te.getTUIAction()

	switch action {
	case createAction:
		createOptions := te.tuiCreate()

		event.Action = action
		event.Name = createOptions[0]
		event.Size = createOptions[1]
		event.Count = createOptions[2]
		event.Version = k8sVersion

		if event.Name == "" {
			reportErr(fmt.Errorf("no cluster name provided"), "set cluster name")
		}

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
			event.Name = te.tuiDelete(clusterList)

			if event.Name == "" {
				os.Exit(0)
			}
		} else {
			fmt.Print(" . No existing clusters to delete\n")

			return Event{Action: exitStatus}
		}

	default:
		fmt.Printf(" ! %s is not a valid Dispatch option\n", event.Action)

		return Event{Action: exitStatus}
	}

	return *event
}

func clusterExists(event Event) bool {
	stackID := event.Name + "-eks"

	clusters := listExistingClusters(event.Bucket)

	for _, cluster := range clusters {
		if strings.Contains(cluster, stackID) {
			return true
		}
	}

	return false
}

func validateClusterName(name string) (bool, error) {
	var err error

	valid := regexp.MustCompile(`^[a-zA-Z][-a-zA-Z0-9]*`).MatchString(name)

	if !valid {
		err = fmt.Errorf("cluster name '%s' is invalid (^[a-zA-Z][-a-zA-Z0-9]*)\ncluster name must begin with a letter", name)
	}

	return valid, err
}
