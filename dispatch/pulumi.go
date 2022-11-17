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

func Exec(event DispatchEvent) {
	// deploy defines AWS resources managed by pulumi
	deploy := func(ctx *pulumi.Context) error {
		eksID := strings.ReplaceAll(event.FQDN, ".", "-")

		// Set cluster values
		minClusterSize, err := strconv.Atoi(event.Count)
		if err != nil {
			reportErr(err, "get cluster node count")
		}
		maxClusterSize := minClusterSize + 2

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

	setPulumiEngine(event.Bucket)
	os.Setenv("PULUMI_CONFIG_PASSPHRASE", "Hello1234")

	ctx := context.Background()

	projectID := event.User + "-eks"
	stackID := strings.ReplaceAll(event.FQDN, ".", "-") + "-eks"

	s, err := auto.UpsertStackInlineSource(ctx, stackID, projectID, deploy)

	w := s.Workspace()

	err = w.InstallPlugin(ctx, "aws", "v4.0.0")
	if err != nil {
		reportErr(err, "Failed to install program plugins")
	}

	region, regionSet := os.LookupEnv("AWS_REGION")
	if !regionSet {
		region = "us-east-1"
	}

	s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: region})

	_, err = s.Refresh(ctx)
	if err != nil {
		reportErr(err, "Failed to refresh stack")
	}

	if !event.Verified {
		var valid string

		fmt.Printf(" Pulumi project: %s", projectID)
		fmt.Printf("\n Pulumi stack: %s", stackID)
		fmt.Printf("\n AWS region: %s\n", region)

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
		os.Exit(0)
	default:
		fmt.Println("Unknown pulumi action.")
	}

	os.Exit(0)
}
