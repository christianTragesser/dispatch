package dispatch

import (
	"errors"
	"testing"
	"time"
)

func TestGetCreationDate(t *testing.T) {
	// provide S3 bucket and dispatch cluster name
	// retrieve date information from S3 object metadata
	t.Run("Retrieve cluster creation date", func(t *testing.T) {
		// report cluster age when metadata exists
		t.Run("Cluster metadata exists", func(t *testing.T) {
			now := time.Now()
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

func TestGetNodeSize(t *testing.T) {
	tests := []struct {
		err            error
		event          mockKopsEvent
		expectedReturn string
		name           string
	}{
		{
			name:           "Get 's' EC2 type",
			event:          mockKopsEvent{size: "s"},
			expectedReturn: smallEC2,
		},
		{
			name:           "Get 'medium' EC2 type",
			event:          mockKopsEvent{size: "medium"},
			expectedReturn: mediumEC2,
		},
		{
			name:           "Get 'Large' EC2 type",
			event:          mockKopsEvent{size: "Large"},
			expectedReturn: largeEC2,
		},
		{
			name:           "Get invalid EC2 type",
			event:          mockKopsEvent{size: "test"},
			expectedReturn: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nodeSize, _ := getNodeSize(test.event.size)

			if nodeSize != test.expectedReturn {
				t.Errorf("getNodeSize unit test failure\n got: '%v', want: '%v'", nodeSize, test.expectedReturn)
			}
		})
	}
}

func TestClusterExists(t *testing.T) {
	tests := []struct {
		list           []string
		event          string
		expectedReturn bool
		name           string
	}{
		{
			name:           "cluster exists",
			event:          "test",
			list:           []string{"no", "nope", "test", "maybe"},
			expectedReturn: true,
		},
		{
			name:           "cluster does not exist",
			event:          "test",
			list:           []string{"no", "nope", "maybe", "definitely"},
			expectedReturn: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cluster := clusterExists(test.list, test.event)

			if cluster != test.expectedReturn {
				t.Errorf("clusterExists unit test failure\n got: '%v', want: '%v'", cluster, test.expectedReturn)
			}
		})
	}
}
