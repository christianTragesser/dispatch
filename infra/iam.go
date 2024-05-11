package infra

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func GetClusterRole(ctx *pulumi.Context, user string, eksID string) (*iam.Role, error) {
	eksClusterRole, err := iam.NewRole(ctx, eksID+"-role", &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Sid": "",
						"Effect": "Allow",
						"Principal": {
							"Service": "eks.amazonaws.com"
						},
						"Action": "sts:AssumeRole"
					}
				]
			}`),
		Tags: pulumi.StringMap{
			"Owner":       pulumi.String(user),
			"EKS cluster": pulumi.String(eksID),
			"Created by":  pulumi.String("Dispatch"),
		},
	})
	if err != nil {
		return nil, err
	}

	return eksClusterRole, nil
}

func GetNodeGroupRole(ctx *pulumi.Context, eksID string) (*iam.Role, error) {

	nodeGroupRole, err := iam.NewRole(ctx, eksID+"-nodegroup-role", &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Sid": "",
						"Effect": "Allow",
						"Principal": {
							"Service": "ec2.amazonaws.com"
						},
						"Action": "sts:AssumeRole"
					}
				]
			}`),
		Tags: pulumi.StringMap{
			"Owner":       pulumi.String("dispatch"),
			"EKS cluster": pulumi.String(eksID),
			"Created by":  pulumi.String("Dispatch"),
		},
	})
	if err != nil {
		return nil, err
	}

	return nodeGroupRole, nil
}

// TODO: Create identity provider OIDC config
// https://www.pulumi.com/registry/packages/aws/api-docs/eks/identityproviderconfig/#identityproviderconfigoidc
/*
	oidcARN := eksCluster.Core.OidcProvider().ApplyT(func(oidc *iam.OpenIdConnectProvider) pulumi.StringOutput {
		return oidc.Arn
	}).(pulumi.StringOutput)

	oidcPolicyURL := eksCluster.Core.OidcProvider().ApplyT(func(oidc *iam.OpenIdConnectProvider) pulumi.StringOutput {
		return pulumi.Sprintf("%v:sub", oidc.Url)
	}).(pulumi.StringOutput)

func GetCertManagerRole(ctx *pulumi.Context, user string, eksID string) (*iam.Role, error) {
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
			"Owner":       pulumi.String(user),
			"EKS cluster": pulumi.String(eksID),
			"Created by":  pulumi.String("Dispatch"),
		},
	})
	if err != nil {
		return nil, err
	}

	return certManagerRole, nil
}

func getTestRole(ctx *pulumi.Context, name string) (*iam.Role, error) {
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
		return nil, err
	}
	json0 := string(tmpJSON0)
	testRole, err := iam.NewRole(ctx, name+"_role", &iam.RoleArgs{
		Name:             pulumi.String("test_role"),
		AssumeRolePolicy: pulumi.String(json0),
		Tags: pulumi.StringMap{
			"tag-key": pulumi.String("tag-value"),
		},
	})
	if err != nil {
		return nil, err
	}

	return testRole, nil
}
*/
