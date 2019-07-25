# Dispatch  
[![pipeline status](https://gitlab.com/christianTragesser/dispatch/badges/master/pipeline.svg)](https://gitlab.com/christianTragesser/dispatch/commits/master)  
A CLI utility for deploying [KOPS](https://github.com/kubernetes/kops) [Kubernetes](https://kubernetes.io/) in public cloud. Dispatch walks you through creating and deleting kubernetes clusters securely without previous cloud or kubernetes experience needed.

### Dependencies
* [Temporary AWS credentials](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html) with access equivalent to managed IAM policies `AmazonEC2FullAccess`, `AmazonRoute53FullAccess`, `AmazonS3FullAccess`, and `AmazonVPCFullAccess`
* Docker
* kubectl installed locally(kubectl config resides at `~/.kube`)

### Initialization
Run a container instance to get started:
```
docker run --rm -it registry.gitlab.com/christiantragesser/dispatch
```
