package dispatch

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	status "github.com/christiantragesser/dispatch/status"
)

const (
	smallEC2  string = "t2.medium"
	mediumEC2 string = "t2.xlarge"
	largeEC2  string = "m4.2xlarge"
)

type KopsEvent struct {
	action, bucket, count, name, size, user, version string
	verify                                           bool
}

func getNodeSize(size string) string {
	var ec2Instance string
	switch size {
	case "small":
		ec2Instance = smallEC2
	case "medium":
		ec2Instance = mediumEC2
	case "large":
		ec2Instance = largeEC2
	default:
		fmt.Print(" ! Invalid EC2 instance size")
		os.Exit(1)
	}

	return ec2Instance
}

func RunKOPS(event KopsEvent) {
	var kopsCMD *exec.Cmd

	home, _ := os.LookupEnv("HOME")

	err := os.Setenv("KUBECONFIG", home+"/.dispatch/.kube/config")
	if err != nil {
		reportErr(err, "set KUBECONFIG environment variable")
	}

	switch event.action {
	case "create":
		zones := getZones()
		nodeSize := getNodeSize(event.size)
		labels := "owner=" + event.user + ", CreatedBy=Dispatch"

		kopsCMD = exec.Command(
			"kops", "create", "cluster",
			"--kubernetes-version="+event.version,
			"--state=s3://"+event.bucket,
			"--node-count="+event.count,
			"--node-size="+nodeSize,
			"--cloud-labels="+labels,
			"--name="+event.name,
			"--zones="+zones,
			"--ssh-public-key=~/.dispatch/.ssh/kops_rsa.pub",
			"--topology=private",
			"--networking=weave",
			"--authorization=RBAC",
			"--yes",
		)

		fmt.Printf("\n"+`
Create cluster details
  - name: %s
  - kubernetes version: %s
  - size: %s
  - nodes: %s`+"\n", event.name, event.version, event.size, event.count)

	case "delete":
		kopsCMD = exec.Command(
			"kops", "delete", "cluster",
			"--name="+event.name,
			"--state=s3://"+event.bucket,
			"--yes",
		)
	default:
		fmt.Print(" ! Unknown KOPS event\n")
		os.Exit(1)
	}

	if !event.verify {
		var valid string

		fmt.Printf("\n ? %s cluster %s (y/n): ", event.action, event.name)
		fmt.Scanf("%s", &valid)

		if valid != "Y" && valid != "y" {
			os.Exit(0)
		}
	}

	fmt.Printf("\n\n Performing %s action for cluster %s\n", event.action, event.name)

	stdout, err := kopsCMD.StdoutPipe()
	if err != nil {
		reportErr(err, "display KOPS stdout")
	}

	if err := kopsCMD.Start(); err != nil {
		reportErr(err, "start KOPS command")
	}

	status.Bar()

	data, err := ioutil.ReadAll(stdout)
	if err != nil {
		reportErr(err, "read KOPS stdout")
	}

	if err := kopsCMD.Wait(); err != nil {
		reportErr(err, "complete KOPS command")
	}

	fmt.Printf("%s\n", string(data))

	if event.action == "create" {
		fmt.Printf("\n Configure your kubectl client for cluster %s with command:\n", event.name)
		fmt.Print("        export KUBECONFIG=\"$HOME/.dispatch/.kube/config\"\n\n", event.name)
	}
}
