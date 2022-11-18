package dispatch

import (
	"fmt"
	"os"

	"github.com/christiantragesser/dispatch/tuiaction"
	"github.com/christiantragesser/dispatch/tuicreate"
	"github.com/christiantragesser/dispatch/tuidelete"
)

const (
	k8sVersion    string = "1.25.3"
	pulumiVersion string = "3.46.1"
	smallEC2      string = "t2.medium"
	mediumEC2     string = "t2.xlarge"
	largeEC2      string = "m4.2xlarge"
	createAction  string = "create"
	deleteAction  string = "delete"
	notFound      string = "not found"
	exitStatus    string = "exit"
	defaultRegion string = "us-east-1"
	defaultScale  int    = 2
)

type Event struct {
	Action   string
	Bucket   string
	Count    string
	FQDN     string
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
	return tuidelete.SelectCluster(clusters)
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

func (e Event) vpcZones() string {
	return getZones()
}

func (e Event) ec2Type(sizeName string) (string, error) {
	return getNodeSize(sizeName)
}

func reportErr(err error, activity string) {
	fmt.Printf(" ! Failed to %s\n\n", activity)
	fmt.Print(err)
	fmt.Print("\n")
	os.Exit(1)
}
