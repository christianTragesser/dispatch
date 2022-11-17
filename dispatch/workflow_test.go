package dispatch

import (
	"os"
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

func ExampleCLIWorkflow_version() {
	var event DispatchEvent

	os.Args[1] = "Version"

	CLIWorkflow("test-Version", event)

	// Output: Dispatch Version test-Version
}

func ExampleCLIWorkflow_help() {
	var event DispatchEvent

	os.Args[1] = "-h"

	CLIWorkflow("help-Version", event)

	// Output: Dispatch options:
	//  dispatch create -h
	//  dispatch delete -h
}

func ExampleCLIWorkflow_deleteHelp() {
	var event DispatchEvent

	os.Args = []string{"dispatch", "delete"}

	CLIWorkflow("delete-Version", event)

	// Output:  ! cluster FQDN is required
}

func ExampleCLIWorkflow_notValid() {
	var event DispatchEvent

	os.Args[1] = "none"

	CLIWorkflow("none-Version", event)

	// Output:  ! none is not a valid Dispatch option
	//
	//  dispatch create -h or dispatch delete -h
}

func ExampleTUIWorkflow_notValid() {
	teAPI := mockTUIEvent{}
	testEvent := DispatchEvent{Action: "test"}

	TUIWorkflow(teAPI, testEvent)

	// Output:  ! test is not a valid Dispatch option
}
