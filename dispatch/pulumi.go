package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/iam"
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

func getExportValue(export map[string]interface{}, field string) string {
	resource := make(map[string]string)

	for k, v := range export {
		switch v.(type) {
		case string:
			resource[k] = fmt.Sprintf("%v", v)
		default:
			reportErr(nil, "determine type")
		}
	}

	return resource[field]
}

func Exec(event *Event) {
	// deploy defines AWS resources managed by pulumi
	deploy := func(ctx *pulumi.Context) error {
		eksID := strings.ReplaceAll(event.Name, ".", "-")

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
			// OIDC provider for IAM RBAC
			CreateOidcProvider: pulumi.BoolPtr(true),
			// Do not give the worker nodes a public IP address
			NodeAssociatePublicIpAddress: pulumi.BoolRef(false),
			Tags: pulumi.StringMap{
				"Owner":       pulumi.String(event.User),
				"EKS cluster": pulumi.String(eksID),
				"Created by":  pulumi.String("Dispatch"),
			},
		})
		if err != nil {
			reportErr(err, "create EKS cluster")
		}

		oidcARN := eksCluster.Core.OidcProvider().ApplyT(func(oidc *iam.OpenIdConnectProvider) pulumi.StringOutput {
			return oidc.Arn
		}).(pulumi.StringOutput)

		oidcPolicyURL := eksCluster.Core.OidcProvider().ApplyT(func(oidc *iam.OpenIdConnectProvider) pulumi.StringOutput {
			return pulumi.Sprintf("%v:sub", oidc.Url)
		}).(pulumi.StringOutput)

		// cert-manager IRSA
		// cert-manager role trust policy
		certManagerTrustPolicy := iam.GetPolicyDocumentOutput(ctx, iam.GetPolicyDocumentOutputArgs{
			Statements: iam.GetPolicyDocumentStatementArray{
				iam.GetPolicyDocumentStatementArgs{
					Sid:    pulumi.String(""),
					Effect: pulumi.String("Allow"),
					Principals: iam.GetPolicyDocumentStatementPrincipalArray{
						iam.GetPolicyDocumentStatementPrincipalArgs{
							Type:        pulumi.String("Federated"),
							Identifiers: pulumi.ToStringArrayOutput([]pulumi.StringOutput{oidcARN}),
						},
					},
					Actions: pulumi.ToStringArrayOutput([]pulumi.StringOutput{pulumi.Sprintf("sts:AssumeRoleWithWebIdentity")}),
					Conditions: iam.GetPolicyDocumentStatementConditionArray{
						iam.GetPolicyDocumentStatementConditionArgs{
							Test:     pulumi.String("StringEquals"),
							Variable: oidcPolicyURL,
							Values:   pulumi.ToStringArrayOutput([]pulumi.StringOutput{pulumi.Sprintf("system:serviceaccount:cert-manager:cert-manager")}),
						},
					},
				},
			},
		})

		// cert-manager Role
		certManagerRole, err := iam.NewRole(ctx, eksID+"-cert-manager", &iam.RoleArgs{
			AssumeRolePolicy: certManagerTrustPolicy.Json(),
			Tags: pulumi.StringMap{
				"Owner":       pulumi.String(event.User),
				"EKS cluster": pulumi.String(eksID),
				"Created by":  pulumi.String("Dispatch"),
			},
		})
		if err != nil {
			reportErr(err, "create cert-manager IAM assume role")
		}

		// ACME DNS01 policy for cert-manager role
		acmeDNS01PolicyJSON, err := json.Marshal(map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []map[string]interface{}{
				{
					"Effect": "Allow",
					"Action": []string{
						"route53:GetChange",
					},
					"Resource": "arn:aws:route53:::change/*",
				},
				{
					"Effect": "Allow",
					"Action": []string{
						"route53:ChangeResourceRecordSets",
						"route53:ListResourceRecordSets",
					},
					"Resource": "arn:aws:route53:::hostedzone/*",
				},
				{
					"Effect": "Allow",
					"Action": []string{
						"route53:ListHostedZonesByName",
					},
					"Resource": "*",
				},
			},
		})
		if err != nil {
			reportErr(err, "create cert-manager inline policy")
		}

		acmePolicyString := string(acmeDNS01PolicyJSON)

		_, err = iam.NewRolePolicy(ctx, eksID+"-acme-dns01", &iam.RolePolicyArgs{
			Role:   certManagerRole.Name,
			Policy: pulumi.String(acmePolicyString),
		})
		if err != nil {
			reportErr(err, "create ACME DNS01 policy")
		}

		if event.Action == createAction {
			ctx.Export("cluster", eksCluster.Core.Cluster())
			ctx.Export("cert-manager-role-arn", certManagerRole.Arn)
		}

		return nil
	}

	if event.Action == deleteAction {
		if !clusterExists(*event) {
			fmt.Printf("\n %s was not found, exiting.\n\n", event.Name)
			os.Exit(0)
		}
	}

	setPulumiEngine(event.Bucket)
	os.Setenv("PULUMI_CONFIG_PASSPHRASE", "Hello1234")

	ctx := context.Background()

	projectID := event.User + "-dispatch"
	stackID := event.Name + "-eks"

	s, err := auto.UpsertStackInlineSource(ctx, stackID, projectID, deploy)
	if err != nil {
		reportErr(err, "create inline source")
	}

	w := s.Workspace()

	err = w.InstallPlugin(ctx, "aws", "v5.21.1")
	if err != nil {
		reportErr(err, "install pulumi plugins")
	}

	region := setAWSRegion()

	if err := s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: region}); err != nil {
		reportErr(err, "set pulumi config")
	}

	_, err = s.Refresh(ctx)
	if err != nil {
		reportErr(err, "to refresh stack")
	}

	if !event.Verified {
		var approve string

		fmt.Printf("\n Cluster name: %s\n", event.Name)

		if event.Action == createAction {
			fmt.Printf(" Cluster node size: %s\n", event.Size)
			fmt.Printf(" Cluster node count: %s\n", event.Count)
		}

		fmt.Printf(" AWS region: %s\n", region)
		fmt.Printf(" Pulumi project: %s\n", projectID)
		fmt.Printf(" Pulumi stack: %s\n", stackID)

		fmt.Printf("\n ? %s cluster %s (y/n): ", event.Action, event.Name)
		fmt.Scanf("%s", &approve)

		if approve != "Y" && approve != "y" {
			os.Exit(0)
		}
	}

	switch event.Action {
	case "create":
		stdoutStreamer := optup.ProgressStreams(os.Stdout)

		res, err := s.Up(ctx, stdoutStreamer)
		if err != nil {
			reportErr(err, "to update stack.")
		}

		expCluster := res.Outputs["cluster"].Value.(map[string]interface{})

		clusterID := getExportValue(expCluster, "id")

		kubeConfigPath := setEKSConfig(clusterID, event.Name)

		fmt.Printf("\n Run the following command for kubectl access to EKS cluster %s:\n", event.Name)
		fmt.Printf(" export KUBECONFIG='%s'\n\n", kubeConfigPath)
	case "delete":
		// wire up our destroy to stream progress to stdout
		stdoutStreamer := optdestroy.ProgressStreams(os.Stdout)

		// destroy our stack and exit early
		_, err := s.Destroy(ctx, stdoutStreamer)
		if err != nil {
			fmt.Printf("Failed to destroy stack: %v", err)
		}

		fmt.Printf("%s stack successfully destroyed\n", stackID)

		if err := w.RemoveStack(ctx, stackID); err != nil {
			reportErr(err, "remove stack")
		}

		fmt.Printf(" - stack %s removed from S3 backend state\n", stackID)
	default:
		fmt.Println("Unknown pulumi action.")
	}
}
