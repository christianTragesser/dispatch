package dispatch

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	pulumiVersion    string = "3.115.0"
	smallEC2         string = "t2.medium"
	mediumEC2        string = "t2.xlarge"
	largeEC2         string = "m4.2xlarge"
	createAction     string = "create"
	deleteAction     string = "delete"
	notFound         string = "not found"
	exitStatus       string = "exit"
	defaultRegion    string = "us-east-1"
	defaultScale     int    = 2
	pulumiStacksPath string = ".pulumi/stacks/"
)

var log = getLogger()

func getLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

type Instance struct {
	Action   string
	Bucket   string
	Count    string
	Name     string
	Size     string
	Verified bool
	Home     workspace
}

func (i Instance) SetWorkspace() (workspace, error) {
	dw, err := i.Home.createWorkspace()
	if err != nil {
		return i.Home, err
	}

	return dw, nil
}

func (i Instance) TestCredentials() error {
	clientConfig := awsClientConfig()

	err := testIAM(clientConfig)
	if err != nil {
		log.Error("Provided credentials do not have permissions needed to run dispatch.")
		return err
	}

	fmt.Printf(" . Valid AWS credentials have been provided for region %s\n", clientConfig.Region)

	return nil

}

func (i Instance) SetBucket() (string, error) {
	clientConfig := awsClientConfig()

	accountNumber, err := getAccountNumber(clientConfig)
	if err != nil {
		return "", err
	}

	uid, err := getDispatchUserID(i.Home.root + "/dispatch.conf")
	if err != nil {
		return "", err
	}

	bucketName := uid + "-dispatch-state-store-" + accountNumber

	existingBuckets, err := listS3Buckets(*clientConfig)
	if err != nil {
		return "", err
	}

	for b := range existingBuckets.Buckets {
		if *existingBuckets.Buckets[b].Name == bucketName {
			fmt.Printf(" . Using s3://%s for dispatch state store\n", bucketName)

			return bucketName, nil
		}
	}

	if i.Bucket == "" {
		var createBucket string

		if !i.Verified {
			fmt.Printf(" ! S3 bucket %s for stack state does not exists\n", bucketName)
			fmt.Printf("\n ? Create S3 bucket %s (y/n): ", bucketName)
			fmt.Scanf("%s", &createBucket)
		}

		if createBucket == "y" || createBucket == "Y" || i.Verified {
			fmt.Println(" + Creating S3 bucket for dispatch state store")
			createStateBucket(*clientConfig, bucketName)
		} else {
			fmt.Print("\n S3 bucket is required for cluster provisioning, exiting.\n\n")
			os.Exit(0)
		}
	}

	return bucketName, nil
}

func (i Instance) ListExistingClusters() error {
	var clusters []string
	clientConfig := awsClientConfig()

	s3Client := s3.NewFromConfig(*clientConfig)

	listConfig := &s3.ListObjectsV2Input{
		Bucket: &i.Bucket,
		Prefix: aws.String(pulumiStacksPath),
	}

	objects, err := s3Client.ListObjectsV2(context.TODO(), listConfig)
	if err != nil {
		log.Error("Failed to list items in dispatch state store.")
		return err
	}

	if len(objects.Contents) > 0 {
		for _, item := range objects.Contents {
			if !strings.Contains(*item.Key, ".bak") {
				clusters = append(clusters, *item.Key)
			}
		}
	} else {
		fmt.Println(" . No existing clusters found")
		return nil
	}

	fmt.Println(" . Existing clusters:")
	for _, item := range clusters {
		fmt.Printf("\t <> %s \n", item)
	}

	return nil
}
