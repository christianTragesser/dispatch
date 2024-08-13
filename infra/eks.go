package infra

import (
	"strconv"

	sg "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/eks"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	defaultScale int = 2
)

func GetEKS(ctx *pulumi.Context, eksVPC *ec2.Vpc, eksClusterRole *iam.Role, user string, eksID string, sg *sg.SecurityGroup) (*eks.Cluster, error) {
	eksCluster, err := eks.NewCluster(ctx, eksID, &eks.ClusterArgs{
		Name:    pulumi.String(eksID),
		RoleArn: eksClusterRole.Arn,
		VpcConfig: &eks.ClusterVpcConfigArgs{
			PublicAccessCidrs: pulumi.StringArray{
				pulumi.String("0.0.0.0/0"),
			},
			SecurityGroupIds: pulumi.StringArray{
				sg.ID().ToStringOutput(),
			},
			SubnetIds: eksVPC.PublicSubnetIds,
		},
		Tags: pulumi.StringMap{
			"Owner":       pulumi.String(user),
			"EKS cluster": pulumi.String(eksID),
			"Created by":  pulumi.String("Dispatch"),
		},
	})
	if err != nil {
		return nil, err
	}

	return eksCluster, nil
}

func GetClusterNodeGroup(ctx *pulumi.Context, user string, eksID string, eksCluster *eks.Cluster, nodeGroupRole *iam.Role, vpc *ec2.Vpc, nodeCount string, nodeType string) (*eks.NodeGroup, error) {
	minClusterSize, err := strconv.Atoi(nodeCount)
	if err != nil {
		return nil, err
	}

	nodeGroup, err := eks.NewNodeGroup(ctx, eksID+"-node-group", &eks.NodeGroupArgs{
		ClusterName:   eksCluster.Name,
		NodeGroupName: pulumi.String(eksID + "-node-group"),
		NodeRoleArn:   pulumi.StringInput(nodeGroupRole.Arn),
		SubnetIds:     vpc.PublicSubnetIds,
		ScalingConfig: &eks.NodeGroupScalingConfigArgs{
			DesiredSize: pulumi.Int(minClusterSize),
			MinSize:     pulumi.Int(minClusterSize),
			MaxSize:     pulumi.Int(minClusterSize + defaultScale),
		},
		InstanceTypes: pulumi.StringArray{pulumi.String(nodeType)},
		Tags: pulumi.StringMap{
			"Owner":       pulumi.String(user),
			"EKS cluster": pulumi.String(eksID),
			"Created by":  pulumi.String("Dispatch"),
		},
	})
	if err != nil {
		return nil, err
	}

	return nodeGroup, nil
}
