package dispatch

import (
	"fmt"
	"strings"

	ta "github.com/christiantragesser/dispatch/tuiaction"
	tc "github.com/christiantragesser/dispatch/tuicreate"
	td "github.com/christiantragesser/dispatch/tuidelete"
)

func (i Instance) TUIWorkflow() (Instance, error) {
	action := ta.Action()

	switch action {
	case createAction:
		createOptions := tc.Create()

		i.Action = action
		i.Name = createOptions[0]
		i.Size = createOptions[1]
		i.Count = createOptions[2]

		if i.Name == "" {
			log.Error("Failed to set cluster name")
			return i, fmt.Errorf("no cluster name provided")
		}
		return i, nil

	case deleteAction:
		var clusterList []map[string]string

		existingClusters, err := i.getExistingClusters()
		if err != nil {
			return i, err
		}

		if len(existingClusters) > 0 {
			for _, c := range existingClusters {
				metadata, err := getObjectMetadata(i.Bucket, c)
				if err != nil {
					return i, err
				}
				cluster := make(map[string]string)
				cluster["name"] = c
				cluster["date"] = metadata.LastModified.Format("2006-01-02 15:04:05") + " UTC"
				clusterList = append(clusterList, cluster)
			}

			selection := td.SelectCluster(clusterList)
			clusterNamespace := strings.TrimPrefix(selection, pulumiStacksPath)
			nameSplit := strings.Split(clusterNamespace, "/")
			nameFile := nameSplit[len(nameSplit)-1]
			i.Name = strings.TrimSuffix(nameFile, "-eks.json")
			i.Action = action

			if i.Name == "" {
				return i, fmt.Errorf("no cluster name provided")
			}
		} else {
			return i, fmt.Errorf("no existing cluster to delete")
		}

	default:
		fmt.Printf(" ! %s is not a valid Dispatch option\n", i.Action)

		return i, nil
	}

	return i, nil
}
