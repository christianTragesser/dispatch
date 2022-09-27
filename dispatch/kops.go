package dispatch

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/christiantragesser/dispatch/status"
)

type kopsCmdAPI interface {
	ec2Type(sizeName string) (string, error)
	vpcZones() string
}

func kopsEventCmd(kcmd kopsCmdAPI, binPath string, event KopsEvent) (*exec.Cmd, error) {
	var kopsCmd *exec.Cmd

	var err error

	switch event.Action {
	case createAction:
		zones := kcmd.vpcZones()
		labels := "owner=" + event.User + ", CreatedBy=Dispatch"

		ec2InstanceType, err := kcmd.ec2Type(event.Size)
		if err != nil {
			reportErr(err, "get EC2 instance Size")
		}

		kopsCmd = exec.Command(
			binPath, "create", "cluster",
			"--name="+event.FQDN,
			"--kubernetes-version="+event.Version,
			"--cloud=aws",
			"--cloud-labels="+labels,
			"--state=s3://"+event.Bucket,
			"--node-count="+event.Count,
			"--node-size="+ec2InstanceType,
			"--zones="+zones,
			"--ssh-public-key=~/.dispatch/.ssh/kops_rsa.pub",
			"--topology=private",
			"--networking=calico",
			"--authorization=RBAC",
			"--yes",
		)
	case deleteAction:
		kopsCmd = exec.Command(
			binPath, "delete", "cluster",
			"--name="+event.FQDN,
			"--state=s3://"+event.Bucket,
			"--yes",
		)
	default:
		err = errors.New("kops event command not known")
	}

	return kopsCmd, err
}

func RunKOPS(event KopsEvent) {
	var kopsCMD *exec.Cmd

	kcmdAPI := KopsEvent{}

	home, _ := os.LookupEnv("HOME")

	kopsBin := home + "/.dispatch/bin/" + kopsVersion + "/" + runtime.GOOS + "/kops"

	err := os.Setenv("KUBECONFIG", home+"/.dispatch/.kube/config")
	if err != nil {
		reportErr(err, "set KUBECONFIG environment variable")
	}

	switch event.Action {
	case createAction:
		existingClusters := listExistingClusters(event.Bucket)

		if clusterExists(existingClusters, event.FQDN) {
			fmt.Printf("\n ! KOPS cluster %s already exists\n\n", event.FQDN)
			os.Exit(1)
		} else {
			kopsCMD, err = kopsEventCmd(kcmdAPI, kopsBin, event)
			if err != nil {
				reportErr(err, "generate create event command")
			}

			fmt.Printf(`
Create cluster details
  - name: %s
  - kubernetes version: %s
  - size: %s
  - nodes: %s`+"\n", event.FQDN, event.Version, event.Size, event.Count)
		}

	case deleteAction:
		existingClusters := listExistingClusters(event.Bucket)

		if clusterExists(existingClusters, event.FQDN) {
			kopsCMD, err = kopsEventCmd(kcmdAPI, kopsBin, event)
			if err != nil {
				reportErr(err, "generate delete event command")
			}
		} else {
			fmt.Printf("\n ! Unknown KOPS cluster %s\n\n", event.FQDN)
			os.Exit(1)
		}
	default:
		fmt.Print(" ! Unknown KOPS event\n")
		os.Exit(1)
	}

	if !event.Verified {
		var valid string

		fmt.Printf("\n ? %s cluster %s (y/n): ", event.Action, event.FQDN)
		fmt.Scanf("%s", &valid)

		if valid != "Y" && valid != "y" {
			os.Exit(0)
		}
	}

	fmt.Printf("\n\n Performing %s action for cluster %s\n", event.Action, event.FQDN)

	stdout, err := kopsCMD.StdoutPipe()
	if err != nil {
		reportErr(err, "display KOPS stdout")
	}

	if err := kopsCMD.Start(); err != nil {
		reportErr(err, "start KOPS command")
	}

	status.Bar()

	data, err := io.ReadAll(stdout)
	if err != nil {
		reportErr(err, "read KOPS stdout")
	}

	if err := kopsCMD.Wait(); err != nil {
		reportErr(err, "complete KOPS command")
	}

	fmt.Printf("%s\n", string(data))

	if event.Action == createAction {
		fmt.Printf("\n Configure your kubectl client for cluster %s with command:\n", event.FQDN)
		fmt.Print("        export KUBECONFIG=\"$HOME/.dispatch/.kube/config\"\n\n")
	}
}
