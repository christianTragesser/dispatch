package dispatch

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var now = time.Now()
var event KopsEvent

type mockKopsEvent struct {
	event                                    KopsEvent
	bucket, count, fqdn, size, user, version string
	verified                                 bool
}

type mockTUIEvent struct {
	action, fqdn, datestamp string
	createDetails           []string
	clusters                []string
	err                     error
}

func (a mockTUIEvent) getTUIAction() string {
	return a.action
}

func (c mockTUIEvent) tuiCreate() []string {
	return c.createDetails
}

func (c mockTUIEvent) tuiDelete(clusters []cluster) string {
	return c.fqdn
}

func (c mockTUIEvent) getClusters(bucket string) []string {
	return c.clusters
}

func (e mockTUIEvent) getClusterCreationDate(bucket string, cluster string) string {

	if e.err != nil {
		return "not found"
	}

	return e.datestamp
}

func TestTUIWorkflow(t *testing.T) {
	tests := []struct {
		expectedReturn KopsEvent
		name           string
		event          mockTUIEvent
	}{
		{
			name: "create defaults",
			event: mockTUIEvent{
				action:        "create",
				createDetails: []string{"dispatch.k8s.local", "small", "2"},
			},
			expectedReturn: KopsEvent{
				Action:   "create",
				fqdn:     "dispatch.k8s.local",
				size:     "small",
				count:    "2",
				version:  "1.21.10",
				verified: false,
			},
		},
		{
			name: "create large testy.cluster.io",
			event: mockTUIEvent{
				action:        "create",
				createDetails: []string{"testy.cluster.io", "large", "20"},
			},
			expectedReturn: KopsEvent{
				Action:   "create",
				fqdn:     "testy.cluster.io",
				size:     "large",
				count:    "20",
				version:  "1.21.10",
				verified: false,
			},
		},
		{
			name: "delete defaults",
			event: mockTUIEvent{
				action:   "delete",
				fqdn:     "dispatch.k8s.local",
				clusters: []string{"dispatch.k8s.local"},
			},
			expectedReturn: KopsEvent{
				Action:   "delete",
				fqdn:     "dispatch.k8s.local",
				verified: false,
			},
		},
		{
			name: "delete testy.cluster.io",
			event: mockTUIEvent{
				action:   "delete",
				fqdn:     "testy.cluster.io",
				clusters: []string{"dispatch.k8s.local", "testy.cluster.io", "dont.delete.io"},
			},
			expectedReturn: KopsEvent{
				Action:   "delete",
				fqdn:     "testy.cluster.io",
				verified: false,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			teAPI := test.event
			testEvent := TUIWorkflow(teAPI, event)

			if testEvent != test.expectedReturn {
				t.Errorf("TUIOption unit test failure '%s'\n got: '%v'\n want: '%v'", test.name, testEvent, test.expectedReturn)
			}
		})
	}
}

func TestCLIOption(t *testing.T) {
	// provide dispatch subcommand of 'create' or 'delete'
	// return kops event containing subcommand flag values
	tests := []struct {
		expectedReturn KopsEvent
		name           string
		osargs         []string
	}{
		{
			name:   "create defaults",
			osargs: []string{"dispatch", "create"},
			expectedReturn: KopsEvent{
				Action:   "create",
				fqdn:     "dispatch.k8s.local",
				size:     "small",
				count:    "2",
				version:  "1.21.10",
				verified: false,
			},
		},
		{
			name:   "create k8s v1.20.8 large testy.cluster.io yolo",
			osargs: []string{"dispatch", "create", "-fqdn", "testy.cluster.io", "-size", "large", "-nodes", "20", "-version", "1.20.8", "-yolo", "true"},
			expectedReturn: KopsEvent{
				Action:   "create",
				fqdn:     "testy.cluster.io",
				size:     "large",
				count:    "20",
				version:  "1.20.8",
				verified: true,
			},
		},
		{
			name:   "delete defaults",
			osargs: []string{"dispatch", "delete", "-fqdn", "dispatch.k8s.local"},
			expectedReturn: KopsEvent{
				Action:   "delete",
				fqdn:     "dispatch.k8s.local",
				verified: false,
			},
		},
		{
			name:   "delete testy.cluster.io yolo",
			osargs: []string{"dispatch", "delete", "-fqdn", "testy.cluster.io", "-yolo", "true"},
			expectedReturn: KopsEvent{
				Action:   "delete",
				fqdn:     "testy.cluster.io",
				verified: true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			os.Args = test.osargs
			testEvent := CLIOption("test-version", event)

			if testEvent != test.expectedReturn {
				t.Errorf("CLIOption unit test failure '%s'\n got: '%v'\n want: '%v'", test.name, testEvent, test.expectedReturn)
			}
		})
	}

}

func mockGetObjectMetadata(bucket string, cluster string) (*s3.HeadObjectOutput, error) {
	if cluster == "error" {
		return nil, errors.New("S3 API request failed.")
	} else {
		return &s3.HeadObjectOutput{LastModified: &now}, nil
	}
}

func TestGetCreationDate(t *testing.T) {
	// provide S3 bucket and dispatch cluster name
	// retrieve date information from S3 object metadata
	t.Run("Retrieve cluster creation date", func(t *testing.T) {
		// report cluster age when metadata exists
		t.Run("Cluster metadata exists", func(t *testing.T) {
			testEvent := mockTUIEvent{datestamp: now.Format("2006-01-02 15:04:05") + " UTC"}
			creationDate := testEvent.getClusterCreationDate("test", "happy")
			expectedReturn := now.Format("2006-01-02 15:04:05") + " UTC"

			if creationDate != expectedReturn {
				t.Errorf("getCreationDate unit test failure\n got: '%v', want: '%v'", creationDate, expectedReturn)
			}
		})
		t.Run("Cluster metadata does not exist", func(t *testing.T) {
			// report 'not found' if failing to retrieve cluster metadeta
			testEvent := mockTUIEvent{err: errors.New("test error")}
			creationDate := testEvent.getClusterCreationDate("test", "error")
			expectedReturn := "not found"

			if creationDate != expectedReturn {
				t.Errorf("getCreationDate unit test failure\n got: '%v', want: '%v'", creationDate, expectedReturn)
			}
		})
	})
}

func ExampleVersionCLIOption() {
	os.Args[1] = "version"
	CLIOption("test-version", event)

	// Output: Dispatch version test-version
}

func ExampleHelpCLIOption() {
	os.Args[1] = "-h"
	CLIOption("help-version", event)

	// Output: Dispatch options:
	//  dispatch create -h
	//  dispatch delete -h
}

func ExampleNotValidCLIOption() {
	os.Args[1] = "none"
	CLIOption("none-version", event)

	// Output:  ! none is not a valid Dispatch option
	//
	//  dispatch create -h or dispatch delete -h
}

func ExampleCLIOptionDeleteNeed() {
	os.Args = []string{"dispatch", "delete"}

	CLIOption("delete-version", event)

	// Output:  ! cluster FQDN is required
}

func ExampleTUIOptionDeleteNone() {
	teAPI := mockTUIEvent{}
	testEvent := KopsEvent{Action: "test"}

	TUIWorkflow(teAPI, testEvent)

	// Output:  ! test is not a valid Dispatch option
}
