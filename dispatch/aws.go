package dispatch

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func awsClientConfig() *aws.Config {
	region, regionSet := os.LookupEnv("AWS_REGION")
	if !regionSet {
		region = "us-east-1"
	}

	profile, profileSet := os.LookupEnv("AWS_PROFILE")
	if !profileSet {
		profile = "default"
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region), config.WithSharedConfigProfile(profile))
	if err != nil {
		reportErr(err, "load AWS configuration")
	}

	return &cfg
}

func testIAM(clientConfig aws.Config) {
	iamClient := iam.NewFromConfig(clientConfig)

	input := &iam.ListUsersInput{MaxItems: aws.Int32(int32(500))}

	_, err := iamClient.ListUsers(context.TODO(), input)
	if err != nil {
		reportErr(err, "list IAM users")
	}
}

func getS3Buckets(clientConfig aws.Config) *s3.ListBucketsOutput {
	s3Client := s3.NewFromConfig(clientConfig)

	buckets, err := s3Client.ListBuckets(context.TODO(), nil)
	if err != nil {
		reportErr(err, "list S3 buckets")
	}

	return buckets
}

func getZones() string {
	var azs string

	clientConfig := awsClientConfig()
	ec2Client := ec2.NewFromConfig(*clientConfig)

	regionValue := []string{clientConfig.Region}
	location := &ec2types.Filter{Name: aws.String("region-name"), Values: regionValue}
	settingFilter := []ec2types.Filter{*location}
	describeSettings := &ec2.DescribeAvailabilityZonesInput{Filters: settingFilter}

	resp, err := ec2Client.DescribeAvailabilityZones(context.TODO(), describeSettings)
	if err != nil {
		reportErr(err, "describe "+clientConfig.Region+" availability zones")
	}

	for i := range resp.AvailabilityZones {
		if i == 0 {
			azs = azs + *resp.AvailabilityZones[i].ZoneName
		} else {
			azs = azs + "," + *resp.AvailabilityZones[i].ZoneName
		}
	}

	return azs
}

func createKOPSBucket(clientConfig aws.Config, bucketName string) {
	s3Client := s3.NewFromConfig(clientConfig)

	// Create private KOPS bucket
	createSettings := &s3.CreateBucketInput{
		Bucket: &bucketName,
		ACL:    "private",
	}

	if clientConfig.Region != "us-east-1" {
		locationConfig := &s3types.CreateBucketConfiguration{LocationConstraint: s3types.BucketLocationConstraint(clientConfig.Region)}
		createSettings.CreateBucketConfiguration = locationConfig
	}

	_, err := s3Client.CreateBucket(context.TODO(), createSettings)
	if err != nil {
		reportErr(err, "create KOPS S3 bucket")
	}

	// Set bucket encryption
	defEnc := &s3types.ServerSideEncryptionByDefault{SSEAlgorithm: s3types.ServerSideEncryptionAes256}
	rule := s3types.ServerSideEncryptionRule{ApplyServerSideEncryptionByDefault: defEnc}
	rules := []s3types.ServerSideEncryptionRule{rule}
	serverConfig := &s3types.ServerSideEncryptionConfiguration{Rules: rules}
	encryptionSettings := &s3.PutBucketEncryptionInput{
		Bucket:                            &bucketName,
		ServerSideEncryptionConfiguration: serverConfig,
	}

	_, err = s3Client.PutBucketEncryption(context.TODO(), encryptionSettings)
	if err != nil {
		reportErr(err, "encrypt KOPS S3 bucket")
	}

	// Enable bucket versioning
	versionConfig := &s3types.VersioningConfiguration{Status: s3types.BucketVersioningStatusEnabled}
	versionSettings := &s3.PutBucketVersioningInput{
		Bucket:                  &bucketName,
		VersioningConfiguration: versionConfig,
	}

	_, err = s3Client.PutBucketVersioning(context.TODO(), versionSettings)
	if err != nil {
		reportErr(err, "version KOPS S3 bucket")
	}
}

func testAWSCreds(clientConfig aws.Config) {
	testIAM(clientConfig)

	fmt.Printf(" . Valid AWS credentials have been provided for region %s\n", clientConfig.Region)
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

func getClusters(bucket string) []string {
	var clusters []string

	clientConfig := awsClientConfig()

	s3Client := s3.NewFromConfig(*clientConfig)

	listConfig := &s3.ListObjectsV2Input{
		Bucket:    &bucket,
		Delimiter: aws.String("/"),
	}

	objects, err := s3Client.ListObjectsV2(context.TODO(), listConfig)
	if err != nil {
		reportErr(err, "list S3 items in KOPS state store")
	}

	if len(objects.CommonPrefixes) > 0 {
		for _, item := range objects.CommonPrefixes {
			clusters = append(clusters, strings.Trim(*item.Prefix, "/"))
		}
	}

	return clusters
}

func listClusters(bucket string) {
	clusters := getClusters(bucket)

	if len(clusters) > 0 {
		fmt.Print(" - Found existing KOPS clusters:\n")
		for _, item := range clusters {
			fmt.Printf("\t <> %s \n", item)
		}
	} else {
		fmt.Print(" . No existing clusters found\n")
	}
}

func getObjectMetadata(bucket string, cluster string) *s3.HeadObjectOutput {
	clientConfig := awsClientConfig()
	s3Client := s3.NewFromConfig(*clientConfig)

	input := &s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    aws.String(cluster + "/config"),
	}

	metadata, err := s3Client.HeadObject(context.TODO(), input)
	if err != nil {
		metadata = &s3.HeadObjectOutput{LastModified: nil}
	}

	return metadata
}
