package dispatch

import (
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/christiantragesser/dispatch/tuiaction"
	"github.com/christiantragesser/dispatch/tuicreate"
)

type cluster struct {
	name, date string
}

type MetadataFunc func(bucket string, cluster string) (*s3.HeadObjectOutput, error)

func getCreationDate(bucket string, cluster string, metadataFunc MetadataFunc) string {
	metadata, err := metadataFunc(bucket, cluster)

	if err != nil {
		return "not found"
	}

	return metadata.LastModified.Format("2006-01-02 15:04:05") + " UTC"
}

func CLIOption(dispatchVersion string, event KopsEvent) KopsEvent {
	action := os.Args[1]

	switch action {
	case "version":
		fmt.Printf("Dispatch version %s\n", dispatchVersion)
		os.Exit(0)
	case "create":
		createCommand := flag.NewFlagSet("create", flag.ExitOnError)
		createFQDN := createCommand.String("fqdn", "dispatch.k8s.local", "Cluster FQDN")
		createSize := createCommand.String("size", "small", "cluster node size")
		nodeCount := createCommand.String("nodes", "2", "cluster node count")
		createVersion := createCommand.String("version", K8S_VERSION, "Kubernetes version")
		createYOLO := createCommand.Bool("yolo", false, "skip verification prompt for cluster creation")

		createCommand.Parse(os.Args[2:])

		event.action = action
		event.fqdn = *createFQDN
		event.size = *createSize
		event.count = *nodeCount
		event.version = *createVersion
		event.verify = *createYOLO

	case "delete":
		deleteCommand := flag.NewFlagSet("delete", flag.ExitOnError)
		deleteFQDN := deleteCommand.String("fqdn", "", "Cluster FQDN")
		deleteYOLO := deleteCommand.Bool("yolo", false, "skip verification prompt for cluster deletion")

		deleteCommand.Parse(os.Args[2:])

		if *deleteFQDN == "" {
			fmt.Print(" ! cluster FQDN is required\n\n")
			os.Exit(0)
		}

		event.action = action
		event.fqdn = *deleteFQDN
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
		event.fqdn = createInfo[0]
		event.size = createInfo[1]
		event.count = createInfo[2]
		event.version = K8S_VERSION

	case "delete":
		currentClusters := []cluster{}

		clusters := getClusters(event.bucket)

		for _, c := range clusters {
			item := cluster{}
			item.name = c
			item.date = getCreationDate(event.bucket, c, getObjectMetadata)
			currentClusters = append(currentClusters, item)
		}

		if len(currentClusters) > 0 {
			event.fqdn = selectCluster(currentClusters)
			event.action = action
		} else {
			fmt.Print(" . No existing clusters to delete\n")
			os.Exit(0)
		}

	default:
		fmt.Printf(" ! %s is not a valid Dispatch option\n", action)
		os.Exit(1)
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

func reportErr(err error, activity string) {
	fmt.Printf(" ! Failed to %s\n\n", activity)
	fmt.Print(err)
	fmt.Print("\n")
	os.Exit(1)
}
