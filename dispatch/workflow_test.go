package dispatch

import (
	"os"
	"testing"
)

type mockKopsEvent struct {
	size string
}

type mockTUIEvent struct {
	action, fqdn, datestamp string
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
	return e.fqdn
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
	t.Parallel()

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
			t.Parallel()
			teAPI := test.event
			testEvent := TUIWorkflow(teAPI, event)

			if testEvent != test.expectedReturn {
				t.Errorf("TUIOption unit test failure '%s'\n got: '%v'\n want: '%v'", test.name, testEvent, test.expectedReturn)
			}
		})
	}
}

func TestCLIWorkflow(t *testing.T) {
	// provide dispatch subcommand of 'create' or 'delete'
	// return kops event containing subcommand flag values
	t.Parallel()

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
			t.Parallel()
			os.Args = test.osargs
			testEvent := CLIWorkflow("test-version", event)

			if testEvent != test.expectedReturn {
				t.Errorf("CLIWorkflow unit test failure '%s'\n got: '%v'\n want: '%v'", test.name, testEvent, test.expectedReturn)
			}
		})
	}
}

func ExampleCLIWorkflow_version() {
	var event KopsEvent

	os.Args[1] = "version"

	CLIWorkflow("test-version", event)

	// Output: Dispatch version test-version
}

func ExampleCLIWorkflow_help() {
	var event KopsEvent

	os.Args[1] = "-h"

	CLIWorkflow("help-version", event)

	// Output: Dispatch options:
	//  dispatch create -h
	//  dispatch delete -h
}

func ExampleCLIWorkflow_deleteHelp() {
	var event KopsEvent

	os.Args = []string{"dispatch", "delete"}

	CLIWorkflow("delete-version", event)

	// Output:  ! cluster FQDN is required
}

func ExampleCLIWorkflow_notValid() {
	var event KopsEvent

	os.Args[1] = "none"

	CLIWorkflow("none-version", event)

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
