# Dispatch  
![CI](https://github.com/christianTragesser/dispatch/actions/workflows/ci.yml/badge.svg) ![Release](https://github.com/christianTragesser/dispatch/actions/workflows/release.yml/badge.svg)  
A CLI utility for deploying [AWS EKS clusters](https://aws.amazon.com/eks/).  
Dispatch simplifies secure, scalable and resilient management of ephemeral kubernetes clusters in AWS.  


### Dependencies
* AWS credentials associated with the following IAM policies:
  - `AmazonEC2FullAccess`
  - `AmazonS3FullAccess`
  - `IAMFullAccess`
  - `AmazonVPCFullAccess`
* [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html) - Access to Dispatch provisioned clusters relies on AWS [Identity and Access Management (IAM)](https://aws.amazon.com/iam/).  
The subcommand `aws eks` is required for initial access to newly provisioned EKS clusters. 
* [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)


#### AWS Authentication
AWS authentication is performed using environment variables or AWS credentials file (`~/.aws/credentials`).  
Environment variable settings take precedence over credential file configuration.

To use an AWS profile other than `default` in your AWS credentials file, set the environment variable `AWS_PROFILE` to the profile name
```
export AWS_PROFILE="my-profile"
```

The following environment variables must be configured if the AWS credentials file is not used:
  - `AWS_ACCESS_KEY_ID`
  - `AWS_SECRET_ACCESS_KEY`
  - `AWS_SESSION_TOKEN`([STS session](https://docs.aws.amazon.com/STS/latest/APIReference/welcome.html))  

#### AWS Region

`us-east-1` is supplied as the default AWS region.  To deploy in a different AWS region, set the environment variable `AWS_REGION` to the region name
```
export AWS_REGION="us-west-2"
```

### Install
#### Homebrew Tap (preferred)
```
brew install christiantragesser/tap/dispatch
```

#### Binary (AMD64)
A `dispatch` [binary is available](https://github.com/christianTragesser/dispatch/releases) for the following platforms: 
* Linux
* MacOS  

Download the binary and place in a directory located in your system `$PATH`

#### Container Image
[christiantragesser/dispatch](https://hub.docker.com/repository/docker/christiantragesser/dispatch)

The container image provides a temporary runtime with all Dispatch dependencies.  

### Use
Run `dispatch` to start a provisioning event
```
$ dispatch
```

Ephemeral Dispatch runtime using docker
```
$ docker run --rm -it christiantragesser/dispatch
# export AWS_ACCESS_KEY_ID=....
# export AWS_SECRET_ACCESS_KEY=....
# export AWS_SESSION_TOKEN=....
# dispatch
```

With Docker using the AWS credentials file
```
$ docker run --rm -it -v $HOME:/root christiantragesser/dispatch
# dispatch
```

Providing a local mount to the container's `/root` directory allows for the persistence of Dispatch event and kubeconfig files.

### CLI Arguments
Events can be configured via CLI subcommands
#### Create
```
$ dispatch create -h
Usage of create:
  -name string
    	cluster name (default "")
  -nodes string
    	cluster node count (default "2")
  -size string
    	cluster node size (default "small")
  -version string
    	Kubernetes version (default "1.24")
  -yes
    	skip verification prompt for cluster creation
```
```
$ dispatch -name my-cluster -nodes 10 -size large -yes
```
#### Delete
```
$ dispatch delete -h
Usage of delete:
  -name string
    	cluster name
  -yes
    	skip verification prompt for cluster deletion
```
```
$ dispatch delete -name my-cluster
```