package dispatch

import (
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
)

type DispatchEvent struct {
	Action   string
	Bucket   string
	Count    string
	FQDN     string
	Size     string
	User     string
	Version  string
	Verified bool
}

func (e DispatchEvent) getTUIAction() string {
	return tuiaction.Action()
}

func (e DispatchEvent) tuiCreate() []string {
	return tuicreate.Create()
}

func (e DispatchEvent) tuiDelete(clusters []map[string]string) string {
	return tuidelete.SelectCluster(clusters)
}

func (e DispatchEvent) getClusters(bucket string) []string {
	return listExistingClusters(bucket)
}

func (e DispatchEvent) getClusterCreationDate(bucket string, cluster string) string {
	metadata, err := getObjectMetadata(bucket, cluster)
	if err != nil {
		return notFound
	}

	return metadata.LastModified.Format("2006-01-02 15:04:05") + " UTC"
}

func (e DispatchEvent) vpcZones() string {
	return getZones()
}

func (e DispatchEvent) ec2Type(sizeName string) (string, error) {
	return getNodeSize(sizeName)
}
