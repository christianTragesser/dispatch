# Dispatch
A CLI utility for deploying [KOPS](https://github.com/kubernetes/kops) [Kubernetes](https://kubernetes.io/) in public cloud. Dispatch walks you through creating and deleting kubernetes clusters securely without previous cloud or kubernetes experience needed.

### Dependencies
* AWS Access Key credentials with IAMFullAccess and AmazonS3FullAccess authorization.
* Docker

### Initialization
Dispatch automates the creation of a KOPS admin user with minimum privileges needed for KOPS cluster administration. Run a container instance to get started:
```
docker run --rm -it christiantragesser/dispatch
```