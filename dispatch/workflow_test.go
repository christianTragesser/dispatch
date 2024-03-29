package dispatch

import (
	"os"
	"testing"
)

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
	_ = clusters
	return e.FQDN
}

func (e mockTUIEvent) getClusters(bucket string) []string {
	_ = bucket
	return e.clusters
}

func (e mockTUIEvent) getClusterCreationDate(bucket string, cluster string) string {
	_ = bucket
	_ = cluster

	if e.err != nil {
		return "not found"
	}

	return e.datestamp
}

func TestValidClusterName(t *testing.T) {
	tests := []struct {
		name      string
		testValue string
		expect    bool
	}{
		{
			name:      "Starts with letter",
			testValue: "my-cluster",
			expect:    true,
		},
		{
			name:      "Starts with number",
			testValue: "12345",
			expect:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := validateClusterName(tc.testValue)

			if got != tc.expect {
				t.Errorf("validClusterName unit test failure '%s'\ngot: '%v'\nwant: '%v'", tc.name, got, tc.expect)
			}
		})
	}
}

func ExampleCLIWorkflow_version() {
	event := &Event{}

	os.Args[1] = "version"

	CLIWorkflow("test-Version", event)

	// Output: Dispatch Version test-Version
}

func ExampleCLIWorkflow_help() {
	event := &Event{}

	os.Args[1] = "-h"

	CLIWorkflow("help-Version", event)

	// Output: Dispatch options:
	//  dispatch create -h
	//  dispatch delete -h
}

func ExampleCLIWorkflow_createHelp() {
	event := &Event{}

	os.Args = []string{"dispatch", "create"}

	CLIWorkflow("create", event)

	// Output:  ! create events require the -name flag
}

func ExampleCLIWorkflow_deleteHelp() {
	event := &Event{}

	os.Args = []string{"dispatch", "delete"}

	CLIWorkflow("delete", event)

	// Output:  ! delete events require the -name flag
}

func ExampleCLIWorkflow_notValid() {
	event := &Event{}

	os.Args[1] = "none"

	CLIWorkflow("none-Version", event)

	// Output:  ! none is not a valid Dispatch option
	//
	//  dispatch create -h or dispatch delete -h
}

func ExampleTUIWorkflow_notValid() {
	teAPI := mockTUIEvent{}
	testEvent := &Event{Action: "test"}

	TUIWorkflow(teAPI, testEvent)

	// Output:  ! test is not a valid Dispatch option
}
