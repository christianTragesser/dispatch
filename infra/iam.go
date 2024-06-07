package infra

import (
	"encoding/json"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/eks"
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

func GetCertManagerRole(ctx *pulumi.Context, cluster *eks.Cluster, user string, eksID string) (*iam.Role, error) {
	var certManagerRole *iam.Role
	/*
		oidcIssuerURL := cluster.Identities.ApplyT(func(identities []eks.ClusterIdentity) (pulumi.StringOutput, error) {
			return pulumi.Sprintf("%v:sub", identities[0].Oidcs[0].Issuer), nil
		}).(pulumi.StringOutput)

		clusterIdentity := cluster.Identities.ApplyT(func(identities []eks.ClusterIdentity) (tls.GetCertificateResultOutput, error) {
			return tls.GetCertificateOutput(ctx, tls.GetCertificateOutputArgs{
				Url: pulumi.StringPtr(*identities[0].Oidcs[0].Issuer),
			}, nil), nil
		}).(tls.GetCertificateResultOutput)

		oidcProvider, err := iam.NewOpenIdConnectProvider(ctx, eksID+"-oidc-provider", &iam.OpenIdConnectProviderArgs{
			Url: cluster.Identities.ApplyT(func(identities []eks.ClusterIdentity) (pulumi.StringOutput, error) {
				return pulumi.Sprintf("%v", identities[0].Oidcs[0].Issuer), nil
			}).(pulumi.StringOutput),
			ClientIdLists: pulumi.StringArray{pulumi.String("sts.amazonaws.com")},
			ThumbprintLists: pulumi.StringArray{
				clusterIdentity.ApplyT(func(identity tls.GetCertificateResult) (*string, error) {
					return &identity.Certificates[0].Sha1Fingerprint, nil
				}).(pulumi.StringPtrOutput).ApplyT(func(s *string) pulumi.StringOutput {
					return pulumi.ToOutput(pulumi.StringPtr(*s)).(pulumi.StringOutput)
				}),
			},
		})
		if err != nil {
			return nil, err
		}
	*/
	/*
		oidcIssuerURL := cluster.Identities.Index(pulumi.Int(0)).Oidcs().Index(pulumi.Int(0)).Issuer()
		callerIdentity, err := sts.GetCallerIdentity(ctx)
		if err != nil {
			return err
		}
		accountID := callerIdentity.AccountId

		// Construct the full OIDC issuer URL using the account ID and EKS OIDC issuer URL
		fullIssuerURL := pulumi.All(accountID, oidcIssuerURL).ApplyT(func(args []interface{}) (string, error) {
			accountId, issuerURL := args[0].(string), args[1].(string)
			// In this example, issuerURL is expected to be in the format "https://oidc.eks.<region>.amazonaws.com/id/<cluster-id>"
			// The issuer URL pattern may vary and needs to be verified accordingly.
			return fmt.Sprintf("https://oidc.eks.%s.amazonaws.com/id/%s", accountId, issuerURL), nil
		}).(pulumi.StringOutput)

		// Get the thumbprint for the OIDC provider
		thumbprintData, err := iam.GetOpenIdConnectProviderThumbprint(ctx, &iam.GetOpenIdConnectProviderThumbprintArgs{
			Url: fullIssuerURL,
		})
		if err != nil {
			return err
		}

		certManagerTrustPolicy := iam.GetPolicyDocumentOutput(ctx, iam.GetPolicyDocumentOutputArgs{
			Statements: iam.GetPolicyDocumentStatementArray{
				iam.GetPolicyDocumentStatementArgs{
					Sid:    pulumi.String(""),
					Effect: pulumi.String("Allow"),
					Principals: iam.GetPolicyDocumentStatementPrincipalArray{
						iam.GetPolicyDocumentStatementPrincipalArgs{
							Type:        pulumi.String("Federated"),
							Identifiers: pulumi.ToStringArrayOutput([]pulumi.StringOutput{oidcProvider.Arn}),
						},
					},
					Actions: pulumi.ToStringArrayOutput([]pulumi.StringOutput{pulumi.Sprintf("sts:AssumeRoleWithWebIdentity")}),
					Conditions: iam.GetPolicyDocumentStatementConditionArray{
						iam.GetPolicyDocumentStatementConditionArgs{
							Test:     pulumi.String("StringEquals"),
							Variable: pulumi.StringInput(oidcIssuerURL),
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

	*/
	return certManagerRole, nil
}

func GetACMEPolicy(ctx *pulumi.Context, eksID string, certManagerRole *iam.Role) (*iam.RolePolicy, error) {
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
		return nil, err
	}

	acmePolicyString := string(acmeDNS01PolicyJSON)

	acmePolicy, err := iam.NewRolePolicy(ctx, eksID+"-acme-dns01", &iam.RolePolicyArgs{
		Role:   certManagerRole.Name,
		Policy: pulumi.String(acmePolicyString),
	})
	if err != nil {
		return nil, err
	}

	return acmePolicy, nil
}

/*
	func getCertThumbprint() (string, error) {
		// Replace with the URL of your OIDC Identity Provider's metadata document
		metadataURL := "https://example.com/.well-known/openid-configuration"

		// Send an HTTP GET request to retrieve the metadata document
		resp, err := http.Get(metadataURL)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		// Parse the metadata document and extract the certificate
		var metadata struct {
			JWKSURL string `json:"jwks_uri"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
			log.Fatal(err)
		}

		// Send an HTTP GET request to retrieve the JWKS (JSON Web Key Set) document
		resp, err = http.Get(metadata.JWKSURL)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		// Parse the JWKS document and extract the certificate
		var jwks struct {
			Keys []struct {
				X509Certificate string `json:"x509_certificate"`
			} `json:"keys"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
			return "", err
		}

		// Extract the first certificate from the JWKS document
		certPEM := jwks.Keys[0].X509Certificate
		block, _ := pem.Decode([]byte(certPEM))
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			log.Fatal(err)
		}

		// Get the thumbprint of the certificate
		thumbprint := fmt.Sprintf("%x", cert.FingerprintSHA1)

		return thumbprint, nil
	}
*/
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
