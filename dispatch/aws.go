package dispatch

// AWS SDK utilities

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// create and configure AWS SDK client
func awsClientConfig() *aws.Config {
	var cfg aws.Config

	var err error

	region := setAWSRegion()

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
			fmt.Println(err)
		}
	} else {
		cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))

		if err != nil {
			log.Error("to find AWS credentials")
		}
	}

	return &cfg
}

func setAWSRegion() string {
	region, regionSet := os.LookupEnv("AWS_REGION")
	if !regionSet {
		region = defaultRegion
	}

	return region
}

// list account IAM users
func testIAM(clientConfig *aws.Config) error {
	maxCount := 500
	iamClient := iam.NewFromConfig(*clientConfig)

	input := &iam.ListUsersInput{MaxItems: aws.Int32(int32(maxCount))}

	_, err := iamClient.ListUsers(context.TODO(), input)
	if err != nil {
		return err
	}

	return nil
}

func getAccountNumber(clientConfig *aws.Config) (string, error) {

	input := &sts.GetCallerIdentityInput{}

	stsClient := sts.NewFromConfig(*clientConfig)

	response, err := stsClient.GetCallerIdentity(context.TODO(), input)
	if err != nil {
		log.Error("Failed to get AWS account number.")
		return "", nil
	}

	return *response.Account, nil
}

// list account S3 buckets
func listS3Buckets(clientConfig aws.Config) (*s3.ListBucketsOutput, error) {
	s3Client := s3.NewFromConfig(clientConfig)

	buckets, err := s3Client.ListBuckets(context.TODO(), nil)
	if err != nil {
		log.Error("Failed to list S3 buckets.")
		return nil, err
	}

	return buckets, nil
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
		log.Error("create KOPS S3 bucket")
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
		log.Error("encrypt KOPS S3 bucket")
	}

	// enable bucket versioning
	versionConfig := &s3types.VersioningConfiguration{Status: s3types.BucketVersioningStatusEnabled}
	versionSettings := &s3.PutBucketVersioningInput{
		Bucket:                  &bucketName,
		VersioningConfiguration: versionConfig,
	}

	_, err = s3Client.PutBucketVersioning(context.TODO(), versionSettings)
	if err != nil {
		log.Error("version KOPS S3 bucket")
	}
}

func getObjectMetadata(bucket string, cluster string) (*s3.HeadObjectOutput, error) {
	clientConfig := awsClientConfig()
	s3Client := s3.NewFromConfig(*clientConfig)

	input := &s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    aws.String(cluster),
	}

	return s3Client.HeadObject(context.TODO(), input)
}
