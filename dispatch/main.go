package dispatch

/*
import (
	"strings"

	"github.com/christiantragesser/dispatch/tuiaction"
	"github.com/christiantragesser/dispatch/tuicreate"
	"github.com/christiantragesser/dispatch/tuidelete"
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
*/
