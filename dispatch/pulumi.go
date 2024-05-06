package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"gopkg.in/yaml.v3"
)

type kubeconfigFile struct {
	APIVersion     string              `yaml:"apiVersion"`
	Kind           string              `yaml:"kind"`
	CurrentContext string              `yaml:"current-context"`
	Preferences    map[string]string   `yaml:"preferences"`
	Clusters       []map[string]string `yaml:"clusters"`
	Users          []map[string]string `yaml:"users"`
	Contexts       []map[string]string `yaml:"contexts"`
}

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

	// deploy defines AWS resources managed by pulumi
	deploy := func(ctx *pulumi.Context) error {
		/*
			eksID := strings.ReplaceAll(i.Name, ".", "-")

			// Set cluster values
			minClusterSize, err := strconv.Atoi(i.Count)
			if err != nil {
				log.Error("get cluster node count")
			}

			maxClusterSize := minClusterSize + defaultScale

			eksNodeInstanceType, err := getNodeSize(i.Size)
			if err != nil {
				log.Error("get node instance type")
			}

			vpcNetworkCidr := "10.0.0.0/16"

			// Create a new VPC, subnets, and associated infrastructure
			eksVpc, err := ec2.NewVpc(ctx, eksID, &ec2.VpcArgs{
				EnableDnsHostnames: pulumi.Bool(true),
				CidrBlock:          &vpcNetworkCidr,
				Tags: pulumi.StringMap{
					"Owner":       pulumi.String(i.User),
					"EKS cluster": pulumi.String(eksID),
					"Created by":  pulumi.String("Dispatch"),
				},
			})
			if err != nil {
				log.Error("create AWS VPC")
			}

			// Create a new EKS cluster
			eksCluster, err := eks.NewCluster(ctx, eksID, &eks.ClusterArgs{
				//Version: pulumi.String(k8sVersion),
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
					"Owner":       pulumi.String(i.User),
					"EKS cluster": pulumi.String(eksID),
					"Created by":  pulumi.String("Dispatch"),
				},
			})
			if err != nil {
				log.Error("create EKS cluster")
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
					"Owner":       pulumi.String(i.User),
					"EKS cluster": pulumi.String(eksID),
					"Created by":  pulumi.String("Dispatch"),
				},
			})
			if err != nil {
				log.Error("create cert-manager IAM assume role")
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
				log.Error("create cert-manager inline policy")
			}

			acmePolicyString := string(acmeDNS01PolicyJSON)

			_, err = iam.NewRolePolicy(ctx, eksID+"-acme-dns01", &iam.RolePolicyArgs{
				Role:   certManagerRole.Name,
				Policy: pulumi.String(acmePolicyString),
			})
			if err != nil {
				log.Error("create ACME DNS01 policy")
			}

			if i.Action == createAction {
				ctx.Export("cluster", eksCluster.Core.Cluster())
				ctx.Export("cert-manager-role-arn", certManagerRole.Arn)
			}
		*/
		tmpJSON0, err := json.Marshal(map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []map[string]interface{}{
				{
					"Action": "sts:AssumeRole",
					"Effect": "Allow",
					"Sid":    "",
					"Principal": map[string]interface{}{
						"Service": "ec2.amazonaws.com",
					},
				},
			},
		})
		if err != nil {
			return err
		}
		json0 := string(tmpJSON0)
		_, err = iam.NewRole(ctx, i.Name+"_role", &iam.RoleArgs{
			Name:             pulumi.String("test_role"),
			AssumeRolePolicy: pulumi.String(json0),
			Tags: pulumi.StringMap{
				"tag-key": pulumi.String("tag-value"),
			},
		})
		if err != nil {
			return err
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

	user, err := getDispatchUserID(i.Home.root + "/dispatch.conf")
	if err != nil {
		return eksCertManagerRoleARN, err
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

	err = i.setPulumiEngine()
	if err != nil {
		return eksCertManagerRoleARN, err
	}

	os.Setenv("PULUMI_CONFIG_PASSPHRASE", "Hello1234")
	os.Setenv("PULUMI_SKIP_UPDATE_CHECK", "true")

	ctx := context.Background()

	stackName := auto.FullyQualifiedStackName("organization", projectID, stackID)

	runDeploy := pulumi.RunFunc(deploy)

	s, err := auto.UpsertStackInlineSource(ctx, stackName, projectID, runDeploy)
	if err != nil {
		log.Error("Failed to create workspace")
		return eksCertManagerRoleARN, err
	}

	w := s.Workspace()

	err = w.InstallPlugin(ctx, "aws", "v6.32.0")
	if err != nil {
		log.Error("Failed to install pulumi plugins")
		return eksCertManagerRoleARN, err
	}

	if err := s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: region}); err != nil {
		log.Error("Failed to set pulumi config")
		return eksCertManagerRoleARN, err
	}

	_, err = s.Refresh(ctx)
	if err != nil {
		log.Error("Failed to refresh stack")
		return eksCertManagerRoleARN, err
	}

	switch i.Action {
	case "create":
		stdoutStreamer := optup.ProgressStreams(os.Stdout)

		_, err := s.Up(ctx, stdoutStreamer)
		if err != nil {
			log.Error("Failed to update stack.")
			return eksCertManagerRoleARN, err
		}
		/*
			expCluster := res.Outputs["cluster"].Value.(map[string]interface{})

			clusterID := getExportValue(expCluster, "id")

			kubeConfigPath := setEKSConfig(clusterID, i.Name)

			eksCertManagerRoleARN = res.Outputs["cert-manager-role-arn"].Value.(string)
			fmt.Printf("\n Run the following command for kubectl access to EKS cluster %s:\n", i.Name)
			fmt.Printf(" export KUBECONFIG='%s'\n\n", kubeConfigPath)
		*/
	case "delete":
		// wire up our destroy to stream progress to stdout
		stdoutStreamer := optdestroy.ProgressStreams(os.Stdout)

		// destroy our stack and exit early
		_, err := s.Destroy(ctx, stdoutStreamer)
		if err != nil {
			fmt.Printf("Failed to destroy stack: %v", err)
			return eksCertManagerRoleARN, err
		}

		fmt.Printf("%s stack successfully destroyed\n", stackID)

		if err := w.RemoveStack(ctx, stackID); err != nil {
			log.Error("remove stack")
			return eksCertManagerRoleARN, err
		}

		fmt.Printf(" - stack %s removed from S3 backend state\n", stackID)

		//clearKubeConfig()
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

func clearKubeConfig() {
	home, homeSet := os.LookupEnv("HOME")

	if homeSet {
		configFile := filepath.Join(home, ".dispatch", ".kube", "config")

		_, readErr := os.Stat(configFile)

		if os.IsNotExist(readErr) {
			fmt.Printf("\nkubeconfig (%s) not found\n", configFile)
		} else {
			cleanConfig := kubeconfigFile{
				APIVersion:     "v1",
				Kind:           "Config",
				CurrentContext: "",
				Clusters:       []map[string]string{},
				Contexts:       []map[string]string{},
				Users:          []map[string]string{},
				Preferences:    map[string]string{},
			}

			configData, err := yaml.Marshal(cleanConfig)
			if err != nil {
				log.Error("construct clean kubeconfig")
			}

			writeErr := os.WriteFile(configFile, configData, fs.FileMode(privMode))
			if writeErr != nil {
				log.Error("write clean kubeconfig")
			}
		}
	} else {
		log.Error("$HOME environment variable not found, exiting.\n")
	}
}
