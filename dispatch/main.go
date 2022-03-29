package dispatch

import (
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

type KopsEvent struct {
	Action                                   string
	bucket, count, fqdn, size, user, version string
	verified                                 bool
}

type cluster struct {
	name, date string
}

type MetadataFunc func(bucket string, cluster string) (*s3.HeadObjectOutput, error)

type TUIEventAPI interface {
	create() []string
	delete() string
	options() string
	getClusters() []string
	getClusterCreationDate(md MetadataFunc, cluster string) string
}

func (e KopsEvent) options() KopsEvent {
	e.Action = tuiaction.Action()

	return e
}

func (e KopsEvent) create() KopsEvent {
	createOptions := tuicreate.Create()

	e.fqdn = createOptions[0]
	e.size = createOptions[1]
	e.count = createOptions[2]
	e.version = K8S_VERSION

	return e
}

func (e KopsEvent) delete(clusters []cluster) KopsEvent {
	e.fqdn = selectCluster(clusters)

	return e
}

func (e KopsEvent) getClusters() []string {
	return listExistingClusters(e.bucket)
}

func (e KopsEvent) getClusterCreationDate(md MetadataFunc, cluster string) string {
	metadata, err := md(e.bucket, cluster)

	if err != nil {
		return "not found"
	}

	return metadata.LastModified.Format("2006-01-02 15:04:05") + " UTC"
}
