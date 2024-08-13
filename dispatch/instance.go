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
	pulumiVersion    string = "3.117.0"
	dispatchConfig   string = "/dispatch.conf"
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

var version = "dev-build"
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

func (i Instance) setWorkspace() (workspace, error) {
	dw, err := i.Home.createWorkspace()
	if err != nil {
		return i.Home, err
	}

	return dw, nil
}

func (i Instance) testCredentials() error {
	clientConfig := awsClientConfig()

	err := testIAM(clientConfig)
	if err != nil {
		log.Error("Provided credentials do not have permissions needed to run dispatch.")
		return err
	}

	fmt.Printf(" . Valid AWS credentials have been provided for region %s\n", clientConfig.Region)

	return nil
}

func (i Instance) setBucket() (string, error) {
	clientConfig := awsClientConfig()

	accountNumber, err := getAccountNumber(clientConfig)
	if err != nil {
		return "", err
	}

	uid, err := getDispatchUserID(i.Home.root + dispatchConfig)
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

			_, err := fmt.Scanf("%s", &createBucket)
			if err != nil {
				return "", err
			}
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

func (i Instance) getExistingClusters() ([]string, error) {
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
		return clusters, err
	}

	if len(objects.Contents) > 0 {
		for _, item := range objects.Contents {
			if !strings.Contains(*item.Key, ".bak") {
				clusters = append(clusters, *item.Key)
			}
		}
	} else {
		fmt.Println(" . No existing clusters found")
		return clusters, nil
	}

	return clusters, nil
}

func (i Instance) getEC2Type() (string, error) {
	size := strings.ToUpper(i.Size)

	switch size {
	case "SMALL", "S":
		return smallEC2, nil
	case "MEDIUM", "M":
		return mediumEC2, nil
	case "LARGE", "L":
		return largeEC2, nil
	default:
		return "", fmt.Errorf("invalid node size: %s", size)
	}
}

func (i Instance) InitInstance() (Instance, error) {
	ws, err := i.setWorkspace()
	if err != nil {
		return i, err
	}

	i.Home = ws

	err = i.testCredentials()
	if err != nil {
		return i, err
	}

	i.Bucket, err = i.setBucket()
	if err != nil {
		return i, err
	}

	clusters, err := i.getExistingClusters()
	if err != nil {
		return i, err
	}

	if len(clusters) > 0 {
		fmt.Println(" . Existing clusters:")

		for _, item := range clusters {
			p := strings.Split(item, "/")
			f := p[len(p)-1]
			c := strings.TrimSuffix(f, "-eks.json")
			fmt.Printf("\t -o- %s \n", c)
		}
	} else {
		fmt.Println(" . No existing clusters found")
	}

	return i, nil
}
