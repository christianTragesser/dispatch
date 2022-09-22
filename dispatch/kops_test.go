package dispatch

import (
	"errors"
	"os/exec"
	"reflect"
	"testing"
)

type mockCmdEvent struct {
	binPath string
	sample  KopsEvent
}

func (e mockCmdEvent) ec2Type(sizeName string) (string, error) {
	var err error

	var ec2Type string

	switch sizeName {
	case "small":
		ec2Type = smallEC2
	case "medium":
		ec2Type = mediumEC2
	case "large":
		ec2Type = largeEC2
	default:
		err = errors.New("type not found")
	}

	return ec2Type, err
}

func (e mockCmdEvent) vpcZones() string {
	return "us-test-1,us-test-2"
}

func TestKopsEventCmd(t *testing.T) {
	tests := []struct {
		expectedReturn *exec.Cmd
		name           string
		event          mockCmdEvent
	}{
		{
			name: "create defaults",
			event: mockCmdEvent{
				binPath: "/test/dir/kops",
				sample: KopsEvent{
					Action:  "create",
					user:    "test",
					size:    "small",
					version: k8sVersion,
					bucket:  "test-bucket",
					count:   "2",
					fqdn:    "dispatch.k8s.local",
				},
			},
			expectedReturn: exec.Command(
				"/test/dir/kops", "create", "cluster",
				"--name=dispatch.k8s.local",
				"--kubernetes-version="+k8sVersion,
				"--cloud=aws",
				"--cloud-labels=owner=test, CreatedBy=Dispatch",
				"--state=s3://test-bucket",
				"--node-count=2",
				"--node-size=t2.medium",
				"--zones=us-test-1,us-test-2",
				"--ssh-public-key=~/.dispatch/.ssh/kops_rsa.pub",
				"--topology=private",
				"--networking=calico",
				"--authorization=RBAC",
				"--yes",
			),
		},
		{
			name: "delete defaults",
			event: mockCmdEvent{
				binPath: "/test/dir/kops",
				sample: KopsEvent{
					Action: "delete",
					bucket: "test-bucket",
					fqdn:   "dispatch.k8s.local",
				},
			},
			expectedReturn: exec.Command(
				"/test/dir/kops", "delete", "cluster",
				"--name=dispatch.k8s.local",
				"--state=s3://test-bucket",
				"--yes",
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmdAPI := mockCmdEvent{}
			testEvent, _ := kopsEventCmd(cmdAPI, test.event.binPath, test.event.sample)

			if !reflect.DeepEqual(testEvent.Args, test.expectedReturn.Args) {
				t.Errorf("kopsEventCmd unit test failure '%s'\n got: '%v'\n want: '%v'", test.name, testEvent.Args, test.expectedReturn.Args)
			}
		})
	}
}
