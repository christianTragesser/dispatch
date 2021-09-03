#!/usr/bin/python
# pylint: disable=R1732
'''
dispatch utils
'''
import os
import sys
from subprocess import Popen
import boto3
import botocore
from kops import KopsException


def set_creds(access_key_id, secret_access_key, session_token):
    '''
    Set AWS session credentials
    '''
    session = boto3.Session(
        aws_access_key_id=access_key_id,
        aws_secret_access_key=secret_access_key,
        aws_session_token=session_token
    )

    return session


def get_s3_buckets(session):
    '''
    returns a list of all S3 buckets in session account
    '''
    s3_client = session.client('s3')

    buckets = s3_client.list_buckets()
    name_list = [x['Name'] for x in buckets['Buckets']]

    return name_list


def exercise_creds(session):
    '''
    Simple use of AWS credentials to validate AWS
    credentials provided to dispatch are valid
    '''
    iam_client = session.client('iam')
    s3_client = session.client('s3')

    try:
        iam_client.list_users()
        s3_client.list_buckets()
        print(' . Valid AWS credentials')
    except botocore.exceptions.ClientError as creds_error:
        print(" ! There is an issue with the provided AWS credentials:")
        print(creds_error)
        sys.exit(1)


def must_mount():
    '''
    Mounts $HOME of docker host to /root directory of dispatch instance
    '''
    if os.path.isdir(f"{os.environ['HOME']}/.ssh"):
        print(' . Found local $HOME mount')
    else:
        mount_message = '''
        It is suggested your $HOME directory path be mounted to /root.
        Use the docker command below to preserve cluster configuration:

        docker run --rm -it -v $HOME:/root \\
        registry.gitlab.com/christiantragesser/dispatch
        '''
        print(mount_message)
        sys.exit(1)


def dispatch_ssh_keys(ssh_key_dir):
    '''
    ensure RSA keys exists for EC2 instances
    '''
    ssh_key = ssh_key_dir+'/kops_rsa'

    if not os.path.isfile(ssh_key):
        print(' * KOPS RSA key not found, generating...')
        print(f' + Creating RSA key {ssh_key}')
        process = Popen(['ssh-keygen', '-t', 'rsa',
              '-b', '4096',
              '-q',
              '-N', '',
              '-f', ssh_key
              ])
        process.wait()

        if process.returncode != 0:
            raise KopsException('There was an issue creating SSH keys.\n')


def create_dir(dir_path):
    '''
    ensure directory for dispatch
    '''
    if not os.path.isdir(dir_path):
        os.mkdir(dir_path, mode = 0o777, dir_fd = None)


def dispatch_workspace():
    '''
    prepares dispatch user working directory
    '''
    dispatch_dir = os.environ['HOME']+'/.dispatch'
    ssh_key_dir = f'{dispatch_dir}/.ssh'
    kube_dir = f'{dispatch_dir}/.kube'
    kube_config = f'{kube_dir}/config'

    create_dir(dispatch_dir)
    create_dir(ssh_key_dir)
    dispatch_ssh_keys(ssh_key_dir)
    create_dir(kube_dir)

    if not os.path.isfile(kube_config):
        with open(kube_config, 'w', encoding="utf8"):
            os.utime(kube_config, None)

        os.chmod(kube_config, 0o664)

def kops_deps(session, name):
    '''
    Checks for existing KOPS S3 bucket,
    creates S3 bucket if doesn't exist
    '''
    s3_client = session.client('s3')

    kops_bucket = name+'-dispatch-kops-state-store'

    user_details = {}
    user_details['bucket'] = kops_bucket

    buckets = get_s3_buckets(session)
    if kops_bucket in buckets:
        print(f' . Using s3://{kops_bucket} for KOPS state.')
    else:
        print(f' ! S3 bucket {kops_bucket} for KOPS state does not exist.')
        create_bucket = input(f' ? Create {kops_bucket} bucket(y/n): ') or 'n'

        if create_bucket in ('y', 'Y'):
            print(f' + Creating KOPS state S3 bucket: {kops_bucket}')
            s3_client.create_bucket(ACL='private', Bucket=kops_bucket, )
            s3_client.put_bucket_encryption(
                Bucket=kops_bucket,
                ServerSideEncryptionConfiguration={
                    'Rules': [
                        {
                            'ApplyServerSideEncryptionByDefault': {
                                'SSEAlgorithm': 'AES256',
                            }
                        },
                    ]
                }
            )
            s3_client.put_bucket_versioning(
                Bucket=kops_bucket,
                VersioningConfiguration={'Status': 'Enabled'}
            )
        else:
            print('\nS3 bucket is required for cluster provisioning, exiting.\n')
            sys.exit(0)

    return user_details
