package infra

import (
	"strings"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/eks"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func GetClusterRole(ctx *pulumi.Context, user string, eksID string) (*iam.Role, error) {
	eksClusterRole, err := iam.NewRole(ctx, eksID+"-cluster-role", &iam.RoleArgs{
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

func GetNodeGroupRole(ctx *pulumi.Context, cluster *eks.Cluster, user string, eksID string, accountNumber string) (*iam.Role, error) {
	issuerURL := cluster.Identities.Index(pulumi.Int(0)).Oidcs().Index(pulumi.Int(0)).Issuer().Elem().ToStringOutput()
	oidcID := issuerURL.ApplyT(func(s string) string {
		s, _ = strings.CutPrefix(s, "https://")
		return s
	})

	policy := pulumi.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
		    {
		    	"Sid": "",
		    	"Effect": "Allow",
		    	"Principal": {
		    		"Service": "ec2.amazonaws.com"
		    	},
		    	"Action": "sts:AssumeRole"
		    },
		    {
		        "Effect": "Allow",
		        "Principal": {
		            "Federated": "arn:aws:iam::%s:oidc-provider/%s"
				},
		        "Condition": {
		            "StringEquals": {
		                "%s:aud": "sts.amazonaws.com",
		                "%s:sub": "system:serviceaccount:kube-system:ebs-csi-controller-sa"
		            }
		        },
		        "Action": "sts:AssumeRoleWithWebIdentity"
		    }
	    ]
	}`, accountNumber, oidcID, oidcID, oidcID)

	nodeGroupRole, err := iam.NewRole(ctx, eksID+"-nodegroup-role", &iam.RoleArgs{
		AssumeRolePolicy: policy,
		Tags: pulumi.StringMap{
			"Owner":       pulumi.String(user),
			"EKS cluster": pulumi.String(eksID),
			"Created by":  pulumi.String("Dispatch"),
		},
	})
	if err != nil {
		return nil, err
	}

	return nodeGroupRole, nil
}

func GetCertManagerRole(ctx *pulumi.Context, cluster *eks.Cluster, user string, eksID string, accountNumber string) (*iam.Role, error) {
	issuerURL := cluster.Identities.Index(pulumi.Int(0)).Oidcs().Index(pulumi.Int(0)).Issuer().Elem().ToStringOutput()
	oidcID := issuerURL.ApplyT(func(s string) string {
		s, _ = strings.CutPrefix(s, "https://")
		return s
	})

	rolePolicy := pulumi.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::%s:oidc-provider/%s"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"%s:sub": "system:serviceaccount:cert-manager:cert-manager"
					}
				}
			}
		]
	}`, accountNumber, oidcID, oidcID)

	certManagerRole, err := iam.NewRole(ctx, eksID+"-cert-manager-role", &iam.RoleArgs{
		AssumeRolePolicy: rolePolicy,
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

func AttachACMEPolicy(ctx *pulumi.Context, certManagerRole *iam.Role, user string, eksID string) error {
	route53JSON := pulumi.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
		  {
			"Effect": "Allow",
			"Action": "route53:GetChange",
			"Resource": "arn:aws:route53:::change/*"
		  },
		  {
			"Effect": "Allow",
			"Action": [
			  "route53:ChangeResourceRecordSets",
			  "route53:ListResourceRecordSets"
			],
			"Resource": "arn:aws:route53:::hostedzone/*"
		  },
		  {
			"Effect": "Allow",
			"Action": "route53:ListHostedZonesByName",
			"Resource": "*"
		  }
		]
	}`)

	route53Policy, err := iam.NewPolicy(ctx, eksID+"-route53-policy", &iam.PolicyArgs{
		Policy: route53JSON,
		Tags: pulumi.StringMap{
			"Owner":       pulumi.String(user),
			"EKS cluster": pulumi.String(eksID),
			"Created by":  pulumi.String("Dispatch"),
		},
	})
	if err != nil {
		return err
	}

	_, err = iam.NewRolePolicyAttachment(ctx, eksID+"-route53-policy-attachment", &iam.RolePolicyAttachmentArgs{
		PolicyArn: route53Policy.Arn,
		Role:      certManagerRole.Name,
	})
	if err != nil {
		return err
	}

	return nil
}
