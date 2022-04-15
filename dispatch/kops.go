package dispatch

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"

	status "github.com/christiantragesser/dispatch/status"
)

func kopsCreateCmd(binPath string, event KopsEvent) *exec.Cmd {
	zones := getZones()
	labels := "owner=" + event.user + ", CreatedBy=Dispatch"

	nodeSize, err := getNodeSize(event.size)
	if err != nil {
		reportErr(err, "get EC2 instance size")
	}

	return exec.Command(
		binPath, "create", "cluster",
		"--kubernetes-version="+event.version,
		"--state=s3://"+event.bucket,
		"--node-count="+event.count,
		"--node-size="+nodeSize,
		"--cloud-labels="+labels,
		"--name="+event.fqdn,
		"--zones="+zones,
		"--ssh-public-key=~/.dispatch/.ssh/kops_rsa.pub",
		"--topology=private",
		"--networking=weave",
		"--authorization=RBAC",
		"--yes",
	)
}

func RunKOPS(event KopsEvent) {
	var kopsCMD *exec.Cmd

	home, _ := os.LookupEnv("HOME")

	kopsBin := home + "/.dispatch/bin/" + kopsVersion + "/" + runtime.GOOS + "/kops"

	err := os.Setenv("KUBECONFIG", home+"/.dispatch/.kube/config")
	if err != nil {
		reportErr(err, "set KUBECONFIG environment variable")
	}

	switch event.Action {
	case createAction:
		existingClusters := listExistingClusters(event.bucket)

		if clusterExists(existingClusters, event.fqdn) {
			fmt.Printf("\n ! KOPS cluster %s already exists\n\n", event.fqdn)
			os.Exit(1)
		} else {
			kopsCMD = kopsCreateCmd(kopsBin, event)

			fmt.Printf(`
Create cluster details
  - name: %s
  - kubernetes version: %s
  - size: %s
  - nodes: %s`+"\n", event.fqdn, event.version, event.size, event.count)
		}

	case deleteAction:
		existingClusters := listExistingClusters(event.bucket)

		if clusterExists(existingClusters, event.fqdn) {
			kopsCMD = exec.Command(
				kopsBin, "delete", "cluster",
				"--name="+event.fqdn,
				"--state=s3://"+event.bucket,
				"--yes",
			)
		} else {
			fmt.Printf("\n ! Unknown KOPS cluster %s\n\n", event.fqdn)
			os.Exit(1)
		}

	default:
		fmt.Print(" ! Unknown KOPS event\n")
		os.Exit(1)
	}

	if !event.verified {
		var valid string

		fmt.Printf("\n ? %s cluster %s (y/n): ", event.Action, event.fqdn)
		fmt.Scanf("%s", &valid)

		if valid != "Y" && valid != "y" {
			os.Exit(0)
		}
	}

	fmt.Printf("\n\n Performing %s action for cluster %s\n", event.Action, event.fqdn)

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

	if event.Action == createAction {
		fmt.Printf("\n Configure your kubectl client for cluster %s with command:\n", event.fqdn)
		fmt.Print("        export KUBECONFIG=\"$HOME/.dispatch/.kube/config\"\n\n", event.fqdn)
	}
}
