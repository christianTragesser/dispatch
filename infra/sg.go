package infra

import (
	sg "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	defaultIngressPort int = 80
)

func GetClusterAccessSG(ctx *pulumi.Context, vpc *ec2.Vpc, user string, eksID string) (*sg.SecurityGroup, error) {
	clusterSg, err := sg.NewSecurityGroup(ctx, eksID+"-cluster-sg", &sg.SecurityGroupArgs{
		VpcId: vpc.VpcId.ToStringPtrOutput(),
		Egress: sg.SecurityGroupEgressArray{
			sg.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Ingress: sg.SecurityGroupIngressArray{
			sg.SecurityGroupIngressArgs{
				Protocol:   pulumi.String("tcp"),
				FromPort:   pulumi.Int(defaultIngressPort),
				ToPort:     pulumi.Int(defaultIngressPort),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
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

	return clusterSg, nil
}
