package dispatch

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ec2"
	"github.com/pulumi/pulumi-eks/sdk/go/eks"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func setPulumiEngine(bucket string) {
	fmt.Println("\nPulumi login to S3 backend....")

	region := setAWSRegion()
	os.Setenv("AWS_REGION", region)

	path, pathSet := os.LookupEnv("PATH")
	if !pathSet {
		fmt.Println("$PATH not set")
	}

	home, homeSet := os.LookupEnv("HOME")
	if !homeSet {
		fmt.Println("$HOME not set")
	}

	pulumiPath := filepath.Join(home, ".dispatch", "bin", "pulumi", pulumiVersion, runtime.GOOS, "pulumi")
	os.Setenv("PATH", path+":"+pulumiPath)

	loginCMD := exec.Command("pulumi", "login", "s3://"+bucket)

	stdout, err := loginCMD.StdoutPipe()
	if err != nil {
		reportErr(err, "display pulumi CMD stdout")
	}

	if err := loginCMD.Start(); err != nil {
		reportErr(err, "start pulumi login")
	}

	data, err := io.ReadAll(stdout)
	if err != nil {
		reportErr(err, "read pulumi login stdout")
	}

	if err := loginCMD.Wait(); err != nil {
		reportErr(err, "complete pulumi login")
	}

	fmt.Printf("%s\n", string(data))
}

func Exec(event Event) {
	// deploy defines AWS resources managed by pulumi
	deploy := func(ctx *pulumi.Context) error {
		eksID := strings.ReplaceAll(event.FQDN, ".", "-")

		// Set cluster values
		minClusterSize, err := strconv.Atoi(event.Count)
		if err != nil {
			reportErr(err, "get cluster node count")
		}

		maxClusterSize := minClusterSize + defaultScale

		eksNodeInstanceType, err := getNodeSize(event.Size)
		if err != nil {
			reportErr(err, "get node instance type")
		}

		vpcNetworkCidr := "10.0.0.0/16"

		// Create a new VPC, subnets, and associated infrastructure
		eksVpc, err := ec2.NewVpc(ctx, eksID, &ec2.VpcArgs{
			EnableDnsHostnames: pulumi.Bool(true),
			CidrBlock:          &vpcNetworkCidr,
			Tags: pulumi.StringMap{
				"Owner":       pulumi.String(event.User),
				"EKS cluster": pulumi.String(eksID),
				"Created by":  pulumi.String("Dispatch"),
			},
		})
		if err != nil {
			reportErr(err, "create AWS VPC")
		}

		// Create a new EKS cluster
		eksCluster, err := eks.NewCluster(ctx, eksID, &eks.ClusterArgs{
			Version: pulumi.String(k8sVersion),
			// Put the cluster in the new VPC created earlier
			VpcId: eksVpc.VpcId,
			// Public subnets will be used for load balancers
			PublicSubnetIds: eksVpc.PublicSubnetIds,
			// Private subnets will be used for cluster nodes
			PrivateSubnetIds: eksVpc.PrivateSubnetIds,
			// Cluster settings
			InstanceType:    pulumi.String(eksNodeInstanceType),
			DesiredCapacity: pulumi.Int(minClusterSize),
			MinSize:         pulumi.Int(minClusterSize),
			MaxSize:         pulumi.Int(maxClusterSize),
			// Do not give the worker nodes a public IP address
			NodeAssociatePublicIpAddress: pulumi.Bool(false),
			Tags: pulumi.StringMap{
				"Owner":       pulumi.String(event.User),
				"EKS cluster": pulumi.String(eksID),
				"Created by":  pulumi.String("Dispatch"),
			},
		})
		if err != nil {
			reportErr(err, "create EKS cluster")
		}

		// Export cluster ID
		ctx.Export("cluster", eksCluster.Core.Cluster())

		return nil
	}

	if event.Action == deleteAction {
		if !clusterExists(event) {
			fmt.Printf("\n %s was not found, exiting.\n\n", event.FQDN)
			os.Exit(0)
		}
	}

	setPulumiEngine(event.Bucket)
	os.Setenv("PULUMI_CONFIG_PASSPHRASE", "Hello1234")

	ctx := context.Background()

	projectID := event.User + "-eks"
	stackID := strings.ReplaceAll(event.FQDN, ".", "-") + "-eks"

	s, err := auto.UpsertStackInlineSource(ctx, stackID, projectID, deploy)
	if err != nil {
		reportErr(err, "create inline source")
	}

	w := s.Workspace()

	err = w.InstallPlugin(ctx, "aws", "v4.0.0")
	if err != nil {
		reportErr(err, "install pulumi plugins")
	}

	region := setAWSRegion()

	if err := s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: region}); err != nil {
		reportErr(err, "set pulumi config")
	}

	_, err = s.Refresh(ctx)
	if err != nil {
		reportErr(err, "failed to refresh stack")
	}

	if !event.Verified {
		var valid string

		fmt.Printf("\n Cluster FQDN: %s\n", event.FQDN)
		fmt.Printf(" Cluster node size: %s\n", event.Size)
		fmt.Printf(" Cluster node count: %s\n", event.Count)
		fmt.Printf(" AWS region: %s\n", region)
		fmt.Printf(" Pulumi project: %s\n", projectID)
		fmt.Printf(" Pulumi stack: %s\n", stackID)

		fmt.Printf("\n ? %s cluster %s (y/n): ", event.Action, event.FQDN)
		fmt.Scanf("%s", &valid)

		if valid != "Y" && valid != "y" {
			os.Exit(0)
		}
	}

	switch event.Action {
	case "create":
		stdoutStreamer := optup.ProgressStreams(os.Stdout)

		res, err := s.Up(ctx, stdoutStreamer)
		if err != nil {
			reportErr(err, "Failed to update stack.")
		}

		output := res.Outputs["cluster"].Value.(map[string]interface{})

		cluster := make(map[string]string)

		for k, v := range output {
			switch v.(type) {
			case string:
				cluster[k] = fmt.Sprintf("%v", v)
			default:
				reportErr(nil, "determine type")
			}
		}

		setEKSConfig(cluster["id"], event.FQDN)
	case "delete":
		fmt.Println("Starting stack destroy")

		// wire up our destroy to stream progress to stdout
		stdoutStreamer := optdestroy.ProgressStreams(os.Stdout)

		// destroy our stack and exit early
		_, err := s.Destroy(ctx, stdoutStreamer)
		if err != nil {
			fmt.Printf("Failed to destroy stack: %v", err)
		}

		fmt.Println("Stack successfully destroyed")

		fmt.Printf("Removing %s stack history and configuration.\n", stackID)

		if err := w.RemoveStack(ctx, stackID); err != nil {
			reportErr(err, "remove stack")
		}
	default:
		fmt.Println("Unknown pulumi action.")
	}
}
