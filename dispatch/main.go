package dispatch

import (
	"fmt"
	"log"
	"strings"

	"github.com/christiantragesser/dispatch/tuiaction"
	"github.com/christiantragesser/dispatch/tuicreate"
	"github.com/christiantragesser/dispatch/tuidelete"
)

const (
	pulumiVersion    string = "3.100.0"
	smallEC2         string = "t2.medium"
	mediumEC2        string = "t2.xlarge"
	largeEC2         string = "m4.2xlarge"
	createAction     string = "create"
	deleteAction     string = "delete"
	notFound         string = "not found"
	exitStatus       string = "exit"
	defaultRegion    string = "us-east-1"
	defaultScale     int    = 2
	pulumiStacksPath string = ".pulumi/stacks/"
)

type Event struct {
	Action   string
	Bucket   string
	Count    string
	Name     string
	Size     string
	User     string
	Version  string
	Verified bool
}

func (e Event) getTUIAction() string {
	return tuiaction.Action()
}

func (e Event) tuiCreate() []string {
	return tuicreate.Create()
}

func (e Event) tuiDelete(clusters []map[string]string) string {
	selection := tuidelete.SelectCluster(clusters)
	clusterNamespace := strings.TrimPrefix(selection, pulumiStacksPath)
	nameSplit := strings.Split(clusterNamespace, "/")
	nameFile := nameSplit[len(nameSplit)-1]
	clusterName := strings.TrimSuffix(nameFile, "-eks.json")

	fmt.Println(clusterName)

	return clusterName
}

func (e Event) getClusters(bucket string) []string {
	return listExistingClusters(bucket)
}

func (e Event) getClusterCreationDate(bucket string, cluster string) string {
	metadata, err := getObjectMetadata(bucket, cluster)
	if err != nil {
		return notFound
	}

	return metadata.LastModified.Format("2006-01-02 15:04:05") + " UTC"
}

func reportErr(err error, activity string) {
	fmt.Printf(" ! Failed to %s\n\n", activity)
	log.Fatalln(err)
}
