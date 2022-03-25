package dispatch

import (
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var now = time.Now()

func mockGetObjectMetadata(bucket string, cluster string) (*s3.HeadObjectOutput, error) {
	if cluster == "error" {
		return nil, errors.New("S3 API request failed.")
	} else {
		return &s3.HeadObjectOutput{LastModified: &now}, nil
	}
}

func TestGetCreationDate(t *testing.T) {
	//provide S3 bucket and dispatch cluster name
	//retrieve date information from S3 object metadata
	//report cluster age if metadata exists
	//report 'not found' if failure to retrieve cluster metadeta
	t.Run("Retrieve cluster creation date", func(t *testing.T) {
		t.Run("Cluster metadata exists", func(t *testing.T) {
			creationDate := getCreationDate("test", "happy", mockGetObjectMetadata)
			expectedReturn := now.Format("2006-01-02 15:04:05") + " UTC"

			if creationDate != expectedReturn {
				t.Errorf("getCreationDate unit test failure: returned '%v' - expecting '%v'", creationDate, expectedReturn)
			}
		})
		t.Run("Cluster metadata does not exist", func(t *testing.T) {
			creationDate := getCreationDate("test", "error", mockGetObjectMetadata)
			expectedReturn := "not found"

			if creationDate != expectedReturn {
				t.Errorf("getCreationDate unit test failure: returned '%v' - expecting '%v'", creationDate, expectedReturn)
			}
		})
	})
}
