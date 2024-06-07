package dispatch

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/christiantragesser/dispatch/infra"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func (i Instance) setPulumiEngine() error {
	fmt.Println("\nPulumi login to S3 backend....")

	region := setAWSRegion()
	os.Setenv("AWS_REGION", region)

	path, pathSet := os.LookupEnv("PATH")
	if !pathSet {
		fmt.Println("$PATH not set")
	}

	pulumiPath := filepath.Join(i.Home.pulumiPath, "pulumi")
	os.Setenv("PATH", path+":"+pulumiPath)

	loginCMD := exec.Command("pulumi", "login", "s3://"+i.Bucket)

	stdout, err := loginCMD.StdoutPipe()
	if err != nil {
		log.Error("Failed to display pulumi CMD stdout")
		return err
	}

	if err := loginCMD.Start(); err != nil {
		log.Error("Failed to start pulumi login")
		return err
	}

	data, err := io.ReadAll(stdout)
	if err != nil {
		log.Error("Failed to read pulumi login stdout")
		return err
	}

	if err := loginCMD.Wait(); err != nil {
		log.Error("Failed to complete pulumi login")
		return err
	}

	fmt.Printf("%s\n", string(data))

	return nil
}

func (i Instance) PulumiExec() (string, error) {
	var eksCertManagerRoleARN string
	user, err := getDispatchUserID(i.Home.root + "/dispatch.conf")
	if err != nil {
		return "", err
	}

	// Declares AWS resources managed by pulumi
	deploy := func(ctx *pulumi.Context) error {
		eksID := strings.ReplaceAll(i.Name, ".", "-")

		// Create a new VPC
		eksVPC, err := infra.GetVPC(ctx, user, eksID)
		if err != nil {
			log.Error("Failed to create VPC")
			return err
		}

		// Create an IAM role for the EKS cluster
		eksClusterRole, err := infra.GetClusterRole(ctx, user, eksID)
		if err != nil {
			log.Error("Failed to create EKS cluster role")
			return err
		}

		// Attach EKS policies to the EKS cluster IAM role
		eksPolicies := []string{
			"arn:aws:iam::aws:policy/AmazonEKSServicePolicy",
			"arn:aws:iam::aws:policy/AmazonEKSClusterPolicy",
		}
		for i, eksPolicy := range eksPolicies {
			_, err := iam.NewRolePolicyAttachment(ctx, fmt.Sprintf("rpa-%d", i), &iam.RolePolicyAttachmentArgs{
				PolicyArn: pulumi.String(eksPolicy),
				Role:      eksClusterRole.Name,
			})
			if err != nil {
				log.Error("Failed to attach EKS policies")
				return err
			}
		}

		// Create an EC2 NodeGroup IAM role
		nodeGroupRole, err := infra.GetNodeGroupRole(ctx, eksID)
		if err != nil {
			log.Error("Failed to create node group role")
			return err
		}

		// Attach NodeGroup policies to the EC2 NodeGroup IAM role
		nodeGroupPolicies := []string{
			"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
			"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
			"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
		}
		for i, nodeGroupPolicy := range nodeGroupPolicies {
			_, err := iam.NewRolePolicyAttachment(ctx, fmt.Sprintf("ngpa-%d", i), &iam.RolePolicyAttachmentArgs{
				Role:      nodeGroupRole.Name,
				PolicyArn: pulumi.String(nodeGroupPolicy),
			})
			if err != nil {
				log.Error("Failed to attach node group policies")
				return err
			}
		}

		// Create cluster API access security group
		clusterAccessSG, err := infra.GetClusterAccessSG(ctx, eksVPC)
		if err != nil {
			log.Error("Failed to create cluster access security group")
			return err
		}

		// Create a new EKS cluster
		eksCluster, err := infra.GetEKS(ctx, eksVPC, eksClusterRole, eksID, clusterAccessSG)
		if err != nil {
			log.Error("Failed to create EKS cluster")
			return err
		}

		// Set EKS node instance type
		eksNodeInstanceType, err := i.getEC2Type()
		if err != nil {
			log.Error("get node instance type")
			return err
		}

		// Create a EKS NodeGroup
		_, err = infra.GetClusterNodeGroup(ctx, eksID,
			eksCluster, nodeGroupRole, eksVPC, i.Count, eksNodeInstanceType)
		if err != nil {
			log.Error("Failed to create node group")
			return err
		}
		/*
			// Create cert-manager IAM role
			certManagerRole, err := infra.GetCertManagerRole(ctx, eksCluster, user, eksID)
			if err != nil {
				log.Error("Failed to create cert manager role")
				return err
			}

			// ACME DNS01 policy for cert-manager role
			_, err = infra.GetACMEPolicy(ctx, eksID, certManagerRole)
			if err != nil {
				log.Error("create cert-manager inline policy")
				return err
			}
		*/
		if i.Action == createAction {
			ctx.Export("cluster", eksCluster.ClusterId)
		}

		return nil
	}

	if i.Action == deleteAction {
		exists, _ := i.clusterExists()
		if !exists {
			fmt.Printf("\n %s was not found, exiting.\n\n", i.Name)
			os.Exit(0)
		}
	}

	projectID := user + "-dispatch"
	stackID := i.Name + "-eks"
	region := setAWSRegion()

	if !i.Verified {
		var approve string

		fmt.Printf("\n Cluster name: %s\n", i.Name)

		if i.Action == createAction {
			fmt.Printf(" Cluster node size: %s\n", i.Size)
			fmt.Printf(" Cluster node count: %s\n", i.Count)
		}

		fmt.Printf(" AWS region: %s\n", region)
		fmt.Printf(" Pulumi project: %s\n", projectID)
		fmt.Printf(" Pulumi stack: %s\n", stackID)

		fmt.Printf("\n ? %s cluster %s (y/n): ", i.Action, i.Name)
		fmt.Scanf("%s", &approve)

		if approve != "Y" && approve != "y" {
			os.Exit(0)
		}
	}

	// Setup Pulumi auto API
	err = i.setPulumiEngine()
	if err != nil {
		return "", err
	}

	os.Setenv("PULUMI_CONFIG_PASSPHRASE", "Hello1234")
	os.Setenv("PULUMI_SKIP_UPDATE_CHECK", "true")

	ctx := context.Background()

	stackName := auto.FullyQualifiedStackName("organization", projectID, stackID)

	// Declared Pulumi payload
	runDeploy := pulumi.RunFunc(deploy)

	s, err := auto.UpsertStackInlineSource(ctx, stackName, projectID, runDeploy)
	if err != nil {
		log.Error("Failed to create workspace")
		return "", err
	}

	w := s.Workspace()

	err = w.InstallPlugin(ctx, "aws", "v6.32.0")
	if err != nil {
		log.Error("Failed to install pulumi plugins")
		return "", err
	}

	if err := s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: region}); err != nil {
		log.Error("Failed to set pulumi config")
		return "", err
	}

	_, err = s.Refresh(ctx)
	if err != nil {
		log.Error("Failed to refresh stack")
		return "", err
	}

	switch i.Action {
	case createAction:
		// Wire up create action for progress stream to stdout
		stdoutStreamer := optup.ProgressStreams(os.Stdout)

		// Create the declared Pulumi stack
		_, err := s.Up(ctx, stdoutStreamer)
		if err != nil {
			log.Error("Failed to update stack.")
			return eksCertManagerRoleARN, err
		}

		// Set EKS kubeconfig
		kubeConfigPath, err := i.setEKSConfig()
		if err != nil {
			return "", err
		}

		//eksCertManagerRoleARN = res.Outputs["cert-manager-role-arn"].Value.(string)
		fmt.Printf("\n Run the following command for kubectl access to EKS cluster %s:\n", i.Name)
		fmt.Printf(" export KUBECONFIG='%s'\n\n", kubeConfigPath)

	case deleteAction:
		// Wire up delete action for progress stream to stdout
		stdoutStreamer := optdestroy.ProgressStreams(os.Stdout)

		// Destroy the declared Pulumi stack
		_, err := s.Destroy(ctx, stdoutStreamer)
		if err != nil {
			fmt.Printf("Failed to destroy stack: %v", err)
			return "", err
		}

		fmt.Printf("%s stack successfully destroyed\n", stackID)

		if err := w.RemoveStack(ctx, stackID); err != nil {
			log.Error("remove stack")
			return "", err
		}

		fmt.Printf(" - stack %s removed from S3 backend state\n", stackID)

	default:
		fmt.Println("Unknown pulumi action.")
	}

	return eksCertManagerRoleARN, nil
}

func (i Instance) clusterExists() (bool, error) {
	stackID := i.Name + "-eks"

	clusters, err := i.getExistingClusters()
	if err != nil {
		return false, err
	}

	for _, cluster := range clusters {
		if strings.Contains(cluster, stackID) {
			return true, nil
		}
	}

	return false, nil
}

func (i Instance) setEKSConfig() (string, error) {
	kubeconfigPath := filepath.Join(i.Home.kubeDir, "config")
	os.Setenv("KUBECONFIG", kubeconfigPath)

	region := setAWSRegion()

	cmd := exec.Command(
		"aws", "eks", "update-kubeconfig",
		"--region", region, "--name", i.Name,
		"--alias", i.Name,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error("Failed to display AWS EKS cmd stdout")
		return "", err
	}

	if err := cmd.Start(); err != nil {
		log.Error("Get EKS kubeconfig command failed")
		return "", err
	}

	data, err := io.ReadAll(stdout)
	if err != nil {
		log.Error("Failed to read AWS EKS command stdout")
		return "", err
	}

	if err := cmd.Wait(); err != nil {
		log.Error("Failed to update kubeconfig")
		return "", err
	}

	fmt.Println(string(data))

	return kubeconfigPath, nil
}
