package dispatch

// AWS SDK utilities

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func getNodeSize(size string) (string, error) {
	var ec2Instance string

	nodeSize := strings.ToUpper(size)

	switch nodeSize {
	case "SMALL", "S":
		ec2Instance = smallEC2
	case "MEDIUM", "M":
		ec2Instance = mediumEC2
	case "LARGE", "L":
		ec2Instance = largeEC2
	default:
		return "", fmt.Errorf("invalid node size: %s", size)
	}

	return ec2Instance, nil
}

// create and configure AWS SDK client
func awsClientConfig() *aws.Config {
	var cfg aws.Config

	var err error

	region, regionSet := os.LookupEnv("AWS_REGION")
	if !regionSet {
		region = defaultRegion
	}

	_, envarCredsSet := os.LookupEnv("AWS_ACCESS_KEY_ID")

	if !envarCredsSet {
		profile, profileSet := os.LookupEnv("AWS_PROFILE")

		if !profileSet {
			profile = "default"
		}

		cfg, err = config.
			LoadDefaultConfig(
				context.TODO(), config.WithRegion(region), config.WithSharedConfigProfile(profile),
			)

		if err != nil {
			fmt.Println(" ! Failed to find AWS credentials in env vars or credentials file")
		}
	} else {
		cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))

		if err != nil {
			reportErr(err, "to find AWS credentials")
		}
	}

	return &cfg
}

// list account IAM users
func testIAM(clientConfig aws.Config) {
	maxCount := 500
	iamClient := iam.NewFromConfig(clientConfig)

	input := &iam.ListUsersInput{MaxItems: aws.Int32(int32(maxCount))}

	_, err := iamClient.ListUsers(context.TODO(), input)
	if err != nil {
		reportErr(err, "list IAM users")
	}
}

// list account S3 buckets
func getS3Buckets(clientConfig aws.Config) *s3.ListBucketsOutput {
	s3Client := s3.NewFromConfig(clientConfig)

	buckets, err := s3Client.ListBuckets(context.TODO(), nil)
	if err != nil {
		reportErr(err, "list S3 buckets")
	}

	return buckets
}

// provide list of AWS region availability zones
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
			azs += *resp.AvailabilityZones[i].ZoneName
		} else {
			azs = azs + "," + *resp.AvailabilityZones[i].ZoneName
		}
	}

	return azs
}

func getAccountNumber() string {
	clientConfig := awsClientConfig()

	input := &sts.GetCallerIdentityInput{}

	stsClient := sts.NewFromConfig(*clientConfig)

	response, err := stsClient.GetCallerIdentity(context.TODO(), input)
	if err != nil {
		reportErr(err, "get caller identity")
	}

	return *response.Account
}

// create S3 bucket for provisioning state
func createStateBucket(clientConfig aws.Config, bucketName string) {
	s3Client := s3.NewFromConfig(clientConfig)

	// create private bucket
	createSettings := &s3.CreateBucketInput{
		Bucket: &bucketName,
		ACL:    "private",
	}

	if clientConfig.Region != defaultRegion {
		locationConfig := &s3types.
			CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(clientConfig.Region),
		}
		createSettings.CreateBucketConfiguration = locationConfig
	}

	_, err := s3Client.CreateBucket(context.TODO(), createSettings)
	if err != nil {
		reportErr(err, "create KOPS S3 bucket")
	}

	// set bucket encryption
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

	// enable bucket versioning
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

	accountNumber := getAccountNumber()

	kopsBucket := user + "-dispatch-state-store-" + accountNumber

	buckets := getS3Buckets(clientConfig)

	for i := range buckets.Buckets {
		if *buckets.Buckets[i].Name == kopsBucket {
			fmt.Printf(" . Using s3://%s for provisioning state store\n", kopsBucket)

			bucketExists = true

			break
		}
	}

	if !bucketExists {
		var createBucket string

		fmt.Printf(" ! S3 bucket %s for stack state does not exists\n", kopsBucket)
		fmt.Printf("\n ? Create S3 bucket %s (y/n): ", kopsBucket)
		fmt.Scanf("%s", &createBucket)

		if createBucket == "y" || createBucket == "Y" {
			createStateBucket(clientConfig, kopsBucket)
		} else {
			fmt.Print("\n S3 bucket is required for cluster provisioning, exiting.\n\n")
			os.Exit(0)
		}
	}

	return kopsBucket
}

func listExistingClusters(bucket string) []string {
	var clusters []string

	clientConfig := awsClientConfig()

	s3Client := s3.NewFromConfig(*clientConfig)

	listConfig := &s3.ListObjectsV2Input{
		Bucket: &bucket,
		Prefix: aws.String(".pulumi/stacks/"),
	}

	objects, err := s3Client.ListObjectsV2(context.TODO(), listConfig)
	if err != nil {
		reportErr(err, "list S3 items in KOPS state store")
	}

	if len(objects.Contents) > 0 {
		for _, item := range objects.Contents {
			if !strings.Contains(*item.Key, ".bak") {
				clusters = append(clusters, *item.Key)
			}
		}
	}

	return clusters
}

func printExistingClusters(bucket string) {
	clusters := listExistingClusters(bucket)

	if len(clusters) > 0 {
		fmt.Print(" - Existing stack configurations:\n")

		for _, item := range clusters {
			fmt.Printf("\t <> %s \n", item)
		}
	} else {
		fmt.Print(" . No existing clusters found\n")
	}
}

func getObjectMetadata(bucket string, cluster string) (*s3.HeadObjectOutput, error) {
	clientConfig := awsClientConfig()
	s3Client := s3.NewFromConfig(*clientConfig)

	input := &s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    aws.String(cluster + "/config"),
	}

	return s3Client.HeadObject(context.TODO(), input)
}

func setEKSConfig(clusterID string, fqdn string) {
	home, homeSet := os.LookupEnv("HOME")
	if !homeSet {
		fmt.Println("$HOME not set")
	}

	kubeconfigPath := filepath.Join(home, ".dispatch", ".kube", "config")
	os.Setenv("KUBECONFIG", kubeconfigPath)

	region, regionSet := os.LookupEnv("AWS_REGION")
	if !regionSet {
		region = defaultRegion
	}

	cmd := exec.Command(
		"aws", "eks", "--region", region,
		"update-kubeconfig", "--name", clusterID,
		"--alias", fqdn,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		reportErr(err, "display aws eks cmd stdout")
	}

	if err := cmd.Start(); err != nil {
		reportErr(err, "start aws eks update")
	}

	data, err := io.ReadAll(stdout)
	if err != nil {
		reportErr(err, "read aws eks stdout")
	}

	if err := cmd.Wait(); err != nil {
		reportErr(err, "update kubeconfig")
	}

	fmt.Println(string(data))
}
