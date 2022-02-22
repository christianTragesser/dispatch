# Dispatch  
[![pipeline status](https://gitlab.com/christianTragesser/dispatch/badges/master/pipeline.svg)](https://gitlab.com/christianTragesser/dispatch/commits/master)  
A CLI utility for deploying [KOPS](https://kops.sigs.k8s.io/) [Kubernetes](https://kubernetes.io/) in AWS. Dispatch simplifies secure management of ephemeral kubernetes clusters.

### Dependencies
* [AWS STS tokens](https://docs.aws.amazon.com/STS/latest/APIReference/welcome.html) associated with the following IAM policies:
  - `AmazonEC2FullAccess`
  - `AmazonRoute53FullAccess`
  - `AmazonS3FullAccess`
  - `IAMFullAccess`
  - `AmazonVPCFullAccess`
* [kOps](https://github.com/kubernetes/kops/releases) installed and found in `$PATH`

Alternatively, Dispatch is available as a container image which includes kOps.

#### AWS Credentials
AWS credentials are configured by environment variables (precedence) or the `default` profile in `$HOME/.aws/credentials`.  
The following environment variables must be configured if AWS credentials are not provided by the `default` profile in the credentials file:
  - `AWS_REGION`
  - `AWS_ACCESS_KEY_ID`
  - `AWS_SECRET_ACCESS_KEY`
  - `AWS_SESSION_TOKEN`

### Install
* [Linux](https://gitlab.com/christianTragesser/dispatch/-/jobs/artifacts/master/download?job=publish:linux)
* [MacOS](https://gitlab.com/christianTragesser/aws-access-key-manager/-/jobs/artifacts/master/download?job=publish:macos)

### Use
Run `dispatch` to start a session.
```
$ dispatch
```

With Docker
```
docker run --rm -it \
       -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
       -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
       -e AWS_SESSION_TOKEN=$AWS_SESSION_TOKEN \
       -v $HOME:/root \
       registry.gitlab.com/christiantragesser/dispatch
```

### CLI Parameters
Sessions can also be implemented via CLI subcommands
#### Create
```
$ dispatch create -h
Usage of create:
  -name string
    	cluster name (default "dispatch.k8s.local")
  -nodes string
    	cluster node count (default "2")
  -size string
    	cluster node size (default "small")
  -version string
    	Kubernetes version (default "1.21.9")
  -yolo
    	skip verification prompt for cluster creation
```
```
$ dispatch -name my-cluster -nodes 10 -size large -yolo true
```
#### Delete
```
$ dispatch delete -h
Usage of delete:
  -name string
    	cluster name
  -yolo
    	skip verification prompt for cluster deletion
```
```
$ dispatch delete -name my-cluster
```

#### Docker CLI Parameters
```
docker run --rm -it \
       -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
       -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
       -e AWS_SESSION_TOKEN=$AWS_SESSION_TOKEN \
       -v $HOME:/root \
       registry.gitlab.com/christiantragesser/dispatch \
       dispatch create -name my-cluster -nodes 10 -size large
```