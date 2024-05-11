package infra

import (
	"github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func GetVPC(ctx *pulumi.Context, user string, eksID string) (*ec2.Vpc, error) {
	subnetStrategy := ec2.SubnetAllocationStrategyAuto
	eksVpc, err := ec2.NewVpc(ctx, eksID, &ec2.VpcArgs{
		CidrBlock:      pulumi.StringRef("10.0.0.0/16"),
		SubnetStrategy: &subnetStrategy,
		SubnetSpecs: []ec2.SubnetSpecArgs{
			{
				Name:     pulumi.StringRef(eksID + "-public"),
				Type:     ec2.SubnetTypePublic,
				CidrMask: pulumi.IntRef(24),
			},
		},
		NatGateways: &ec2.NatGatewayConfigurationArgs{
			Strategy: ec2.NatGatewayStrategyNone,
		},
		// provide a NatGateways configuration block
		Tags: pulumi.StringMap{
			"Owner":       pulumi.String(user),
			"EKS cluster": pulumi.String(eksID),
			"Created by":  pulumi.String("Dispatch"),
		},
	})
	if err != nil {
		return nil, err
	}

	return eksVpc, nil
}
