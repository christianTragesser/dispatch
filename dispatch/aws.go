package dispatch

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func awsClientConfig() *aws.Config {
	region, regionSet := os.LookupEnv("AWS_REGION")
	if !regionSet {
		region = "us-east-1"
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		ReportErr(err, "load AWS configuration")
	}

	return &cfg
}

func testIAM(clientConfig aws.Config) {
	iamClient := iam.NewFromConfig(clientConfig)

	input := &iam.ListUsersInput{MaxItems: aws.Int32(int32(500))}

	_, err := iamClient.ListUsers(context.TODO(), input)
	if err != nil {
		ReportErr(err, "list IAM users")
	}
}

func getS3Buckets(clientConfig aws.Config) *s3.ListBucketsOutput {
	s3Client := s3.NewFromConfig(clientConfig)

	buckets, err := s3Client.ListBuckets(context.TODO(), nil)
	if err != nil {
		ReportErr(err, "list S3 buckets")
	}

	return buckets
}

func createKOPSBucket(clientConfig aws.Config, bucketName string) {
	s3Client := s3.NewFromConfig(clientConfig)

	// Create private KOPS bucket
	createSettings := &s3.CreateBucketInput{
		Bucket: &bucketName,
		ACL:    "private",
	}

	if clientConfig.Region != "us-east-1" {
		locationConfig := &types.CreateBucketConfiguration{LocationConstraint: types.BucketLocationConstraint(clientConfig.Region)}
		createSettings.CreateBucketConfiguration = locationConfig
	}

	_, err := s3Client.CreateBucket(context.TODO(), createSettings)
	if err != nil {
		ReportErr(err, "create KOPS S3 bucket")
	}

	// Set bucket encryption
	defEnc := &types.ServerSideEncryptionByDefault{SSEAlgorithm: types.ServerSideEncryptionAes256}
	rule := types.ServerSideEncryptionRule{ApplyServerSideEncryptionByDefault: defEnc}
	rules := []types.ServerSideEncryptionRule{rule}
	serverConfig := &types.ServerSideEncryptionConfiguration{Rules: rules}
	encryptionSettings := &s3.PutBucketEncryptionInput{
		Bucket:                            &bucketName,
		ServerSideEncryptionConfiguration: serverConfig,
	}

	_, err = s3Client.PutBucketEncryption(context.TODO(), encryptionSettings)
	if err != nil {
		ReportErr(err, "encrypt KOPS S3 bucket")
	}

	// Enable bucket versioning
	versionConfig := &types.VersioningConfiguration{Status: types.BucketVersioningStatusEnabled}
	versionSettings := &s3.PutBucketVersioningInput{
		Bucket:                  &bucketName,
		VersioningConfiguration: versionConfig,
	}

	_, err = s3Client.PutBucketVersioning(context.TODO(), versionSettings)
	if err != nil {
		ReportErr(err, "version KOPS S3 bucket")
	}
}

func testAWSCreds(clientConfig aws.Config) {
	fmt.Print("\nEnsuring AWS dependencies:\n")

	_, accessKeySet := os.LookupEnv("AWS_ACCESS_KEY_ID")
	if !accessKeySet {
		fmt.Print("(No AWS env var creds, not sure this will end well....)\n")
	}

	testIAM(clientConfig)
	fmt.Print(" . Valid AWS credentials have been provided\n")
}

func ensureS3Bucket(clientConfig aws.Config, user string) string {
	var bucketExists bool
	kopsBucket := user + "-dispatch-kops-state-store"

	buckets := getS3Buckets(clientConfig)

	for i := range buckets.Buckets {
		if *buckets.Buckets[i].Name == kopsBucket {
			fmt.Printf(" . Using s3://%s for KOPS state\n", kopsBucket)
			bucketExists = true
			break
		}
	}

	if !bucketExists {
		var createBucket string

		fmt.Printf(" ! S3 bucket %s for KOPS state doesn't exists\n", kopsBucket)
		fmt.Printf("\n ? Create S3 bucket %s (y/n): ", kopsBucket)
		fmt.Scanf("%s", &createBucket)

		if createBucket == "y" || createBucket == "Y" {
			createKOPSBucket(clientConfig, kopsBucket)
		} else {
			fmt.Print("\n S3 bucket is required for cluster provisioning, exiting.\n\n")
			os.Exit(0)
		}
	}

	return kopsBucket
}

func EnsureDependencies(userID string) string {
	clientConfig := awsClientConfig()
	testAWSCreds(*clientConfig)

	return ensureS3Bucket(*clientConfig, userID)
}

func ListClusters(bucket string) {
	clientConfig := awsClientConfig()

	s3Client := s3.NewFromConfig(*clientConfig)

	listConfig := &s3.ListObjectsV2Input{
		Bucket:    &bucket,
		Delimiter: aws.String("/"),
	}

	objects, err := s3Client.ListObjectsV2(context.TODO(), listConfig)
	if err != nil {
		ReportErr(err, "list bucket objects")
	}

	if len(objects.CommonPrefixes) > 0 {
		fmt.Print("\n Existing KOPS clusters:\n")
		for _, item := range objects.CommonPrefixes {
			fmt.Printf("\t - %s \n", strings.Trim(*item.Prefix, "/"))
		}
	} else {
		fmt.Print("\n No existing clusters found\n")
	}
}
