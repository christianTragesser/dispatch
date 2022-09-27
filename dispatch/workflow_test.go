package dispatch

import (
	"os"
	"testing"
)

type mockKopsEvent struct {
	Size string
}

type mockTUIEvent struct {
	action, FQDN, datestamp string
	createDetails           []string
	clusters                []string
	err                     error
}

func (e mockTUIEvent) getTUIAction() string {
	return e.action
}

func (e mockTUIEvent) tuiCreate() []string {
	return e.createDetails
}

func (e mockTUIEvent) tuiDelete(clusters []map[string]string) string {
	return e.FQDN
}

func (e mockTUIEvent) getClusters(bucket string) []string {
	return e.clusters
}

func (e mockTUIEvent) getClusterCreationDate(bucket string, cluster string) string {
	if e.err != nil {
		return "not found"
	}

	return e.datestamp
}

func TestTUIWorkflow(t *testing.T) {
	var event KopsEvent

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
				FQDN:     "dispatch.k8s.local",
				Size:     "small",
				Count:    "2",
				Version:  k8sVersion,
				Verified: false,
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
				FQDN:     "testy.cluster.io",
				Size:     "large",
				Count:    "20",
				Version:  k8sVersion,
				Verified: false,
			},
		},
		{
			name: "delete defaults",
			event: mockTUIEvent{
				action:   "delete",
				FQDN:     "dispatch.k8s.local",
				clusters: []string{"dispatch.k8s.local"},
			},
			expectedReturn: KopsEvent{
				Action:   "delete",
				FQDN:     "dispatch.k8s.local",
				Verified: false,
			},
		},
		{
			name: "delete testy.cluster.io",
			event: mockTUIEvent{
				action:   "delete",
				FQDN:     "testy.cluster.io",
				clusters: []string{"dispatch.k8s.local", "testy.cluster.io", "dont.delete.io"},
			},
			expectedReturn: KopsEvent{
				Action:   "delete",
				FQDN:     "testy.cluster.io",
				Verified: false,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			teAPI := test.event
			testEvent := TUIWorkflow(teAPI, event)

			if testEvent != test.expectedReturn {
				t.Errorf("TUIWorkflow unit test failure '%s'\n got: '%v'\n want: '%v'", test.name, testEvent, test.expectedReturn)
			}
		})
	}
}

func TestCLIWorkflow(t *testing.T) {
	// provide dispatch subcommand of 'create' or 'delete'
	// return kops event containing subcommand flag values
	var event KopsEvent

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
				FQDN:     "dispatch.k8s.local",
				Size:     "small",
				Count:    "2",
				Version:  k8sVersion,
				Verified: false,
			},
		},
		{
			name:   "create k8s v1.20.8 large testy.cluster.io yolo",
			osargs: []string{"dispatch", "create", "-fqdn", "testy.cluster.io", "-size", "large", "-nodes", "20", "-version", "1.20.8", "-yolo", "true"},
			expectedReturn: KopsEvent{
				Action:   "create",
				FQDN:     "testy.cluster.io",
				Size:     "large",
				Count:    "20",
				Version:  "1.20.8",
				Verified: true,
			},
		},
		{
			name:   "delete defaults",
			osargs: []string{"dispatch", "delete", "-fqdn", "dispatch.k8s.local"},
			expectedReturn: KopsEvent{
				Action:   "delete",
				FQDN:     "dispatch.k8s.local",
				Verified: false,
			},
		},
		{
			name:   "delete testy.cluster.io yolo",
			osargs: []string{"dispatch", "delete", "-fqdn", "testy.cluster.io", "-yolo", "true"},
			expectedReturn: KopsEvent{
				Action:   "delete",
				FQDN:     "testy.cluster.io",
				Verified: true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			os.Args = test.osargs
			testEvent := CLIWorkflow("test-Version", event)

			if testEvent != test.expectedReturn {
				t.Errorf("CLIWorkflow unit test failure '%s'\n got: '%v'\n want: '%v'", test.name, testEvent, test.expectedReturn)
			}
		})
	}
}

func ExampleCLIWorkflow_version() {
	var event KopsEvent

	os.Args[1] = "Version"

	CLIWorkflow("test-Version", event)

	// Output: Dispatch Version test-Version
}

func ExampleCLIWorkflow_help() {
	var event KopsEvent

	os.Args[1] = "-h"

	CLIWorkflow("help-Version", event)

	// Output: Dispatch options:
	//  dispatch create -h
	//  dispatch delete -h
}

func ExampleCLIWorkflow_deleteHelp() {
	var event KopsEvent

	os.Args = []string{"dispatch", "delete"}

	CLIWorkflow("delete-Version", event)

	// Output:  ! cluster FQDN is required
}

func ExampleCLIWorkflow_notValid() {
	var event KopsEvent

	os.Args[1] = "none"

	CLIWorkflow("none-Version", event)

	// Output:  ! none is not a valid Dispatch option
	//
	//  dispatch create -h or dispatch delete -h
}

func ExampleTUIWorkflow_notValid() {
	teAPI := mockTUIEvent{}
	testEvent := KopsEvent{Action: "test"}

	TUIWorkflow(teAPI, testEvent)

	// Output:  ! test is not a valid Dispatch option
}
