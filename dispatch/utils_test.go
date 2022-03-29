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
			testEvent := KopsEvent{bucket: "test-bucket"}
			creationDate := testEvent.getClusterCreationDate(mockGetObjectMetadata, "happy")
			expectedReturn := now.Format("2006-01-02 15:04:05") + " UTC"

			if creationDate != expectedReturn {
				t.Errorf("getCreationDate unit test failure\n got: '%v', want: '%v'", creationDate, expectedReturn)
			}
		})
		t.Run("Cluster metadata does not exist", func(t *testing.T) {
			// report 'not found' if failing to retrieve cluster metadeta
			testEvent := KopsEvent{bucket: "test-bucket"}
			creationDate := testEvent.getClusterCreationDate(mockGetObjectMetadata, "error")
			expectedReturn := "not found"

			if creationDate != expectedReturn {
				t.Errorf("getCreationDate unit test failure\n got: '%v', want: '%v'", creationDate, expectedReturn)
			}
		})
	})
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

//func TestTUIOption(t *testing.T) {
//	tuiEvent := KopsEvent{bucket: "test"}
//
//	tests := []struct {
//		expectedReturn KopsEvent
//		name           string
//		option         TUIActionFunc
//		createInfo     TUICreateFunc
//		clustersInfo   GetClustersFunc
//	}{
//		{
//			name:         "create defaults",
//			option:       func() string { return "create" },
//			createInfo:   func() []string { return []string{"dispatch.k8s.local", "small", "2"} },
//			clustersInfo: func(bucket string) []string { return []string{} },
//			expectedReturn: KopsEvent{
//				Action:   "create",
//				fqdn:     "dispatch.k8s.local",
//				size:     "small",
//				count:    "2",
//				version:  "1.21.10",
//				verified: false,
//			},
//		},
//		{
//			name:         "create large testy.cluster.io",
//			option:       func() string { return "create" },
//			createInfo:   func() []string { return []string{"testy.cluster.io", "large", "20"} },
//			clustersInfo: func(bucket string) []string { return []string{} },
//			expectedReturn: KopsEvent{
//				Action:   "create",
//				fqdn:     "testy.cluster.io",
//				size:     "large",
//				count:    "20",
//				version:  "1.21.10",
//				verified: false,
//			},
//		},
//		{
//			name:         "delete none",
//			option:       func() string { return "delete" },
//			createInfo:   func() []string { return []string{} },
//			clustersInfo: func(bucket string) []string { return []string{} },
//			expectedReturn: KopsEvent{
//				Action:   "delete",
//				fqdn:     "dispatch.k8s.local",
//				verified: false,
//			},
//		},
//	}
//
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			testEvent := TUIOption(tuiEvent, test.option, test.createInfo, test.clustersInfo)
//
//			if testEvent != test.expectedReturn {
//				t.Errorf("TUIOption unit test failure '%s'\n got: '%v'\n want: '%v'", test.name, testEvent, test.expectedReturn)
//			}
//		})
//	}
//}

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
	testEvent := KopsEvent{Action: "test"}

	TUIWorkflow(testEvent)

	// Output:  ! test is not a valid Dispatch option
}
