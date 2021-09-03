# Dispatch  
[![pipeline status](https://gitlab.com/christianTragesser/dispatch/badges/master/pipeline.svg)](https://gitlab.com/christianTragesser/dispatch/commits/master)  
A CLI utility for deploying [KOPS](https://github.com/kubernetes/kops) [Kubernetes](https://kubernetes.io/) in AWS. Dispatch guides you through creating and deleting kubernetes clusters securely without previous AWS or kubernetes experience needed.

### Dependencies
* [AWS STS tokens](https://docs.aws.amazon.com/STS/latest/APIReference/welcome.html) with an assume role with following managed policies attached:
  - `AmazonEC2FullAccess`
  - `AmazonRoute53FullAccess`
  - `AmazonS3FullAccess`
  - `IAMFullAccess`
  - `AmazonVPCFullAccess`
* Docker

### Initialization
Run a container instance to get started:
```
docker run --rm -it -v $HOME:/root registry.gitlab.com/christiantragesser/dispatch
```

To provision a specific version([supported by KOPS](https://kops.sigs.k8s.io/welcome/releases/)) of Kubernetes, supply the environment variable `K8S_VERSION` when running an instance:
```
docker run --rm -it -e K8S_VERSION='1.22.1' -v $HOME:/root registry.gitlab.com/christiantragesser/dispatch
```

### CLI Tools
`kubectl`, `kops`, and `awscli` command line clients are provided during a _Shell session_. 