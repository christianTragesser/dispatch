package dispatch

import (
	"github.com/christiantragesser/dispatch/tuiaction"
	"github.com/christiantragesser/dispatch/tuicreate"
)

const (
	K8S_VERSION  string = "1.21.10"
	KOPS_VERSION string = "1.21.4"
	smallEC2     string = "t2.medium"
	mediumEC2    string = "t2.xlarge"
	largeEC2     string = "m4.2xlarge"
)

type cluster struct {
	name, date string
}

type KopsEvent struct {
	Action                                   string
	bucket, count, fqdn, size, user, version string
	verified                                 bool
}

type TUIEventAPI interface {
	getTUIAction() string
	tuiCreate() []string
	tuiDelete(cluster []cluster) string
	getClusters(bucket string) []string
	getClusterCreationDate(bucket string, cluster string) string
}

func (e KopsEvent) getTUIAction() string {
	return tuiaction.Action()
}

func (e KopsEvent) tuiCreate() []string {
	return tuicreate.Create()
}

func (e KopsEvent) tuiDelete(clusters []cluster) string {
	return selectCluster(clusters)
}

func (e KopsEvent) getClusters(bucket string) []string {
	return listExistingClusters(bucket)
}

func (e KopsEvent) getClusterCreationDate(bucket string, cluster string) string {
	metadata, err := getObjectMetadata(bucket, cluster)

	if err != nil {
		return "not found"
	}

	return metadata.LastModified.Format("2006-01-02 15:04:05") + " UTC"
}
