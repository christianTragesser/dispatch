#!/usr/bin/python
'''
dispatch utils
'''
import os
import sys
import boto3
import botocore

MANAGED_POLICIES = ['AmazonEC2FullAccess', 'AmazonRoute53FullAccess', 'AmazonS3FullAccess',
                    'IAMFullAccess', 'AmazonVPCFullAccess']
ARN_PREFIX = 'arn:aws:iam::aws:policy/'


#def get_users(session):
#    '''
#
#    '''
#    iam = session.client('iam')
#
#    users = []
#    paginator = iam.get_paginator('list_users')
#    for response in paginator.paginate():
#        for user_name in response['Users']:
#            users.append(user_name['UserName'])
#    return users
#
#
#def get_groups(session):
#    iam = session.client('iam')
#
#    groups = []
#    paginator = iam.get_paginator('list_groups')
#    for response in paginator.paginate():
#        for group_name in response['Groups']:
#            groups.append(group_name['GroupName'])
#    return groups
#
#
#def get_user_groups(session, user):
#    iam = session.client('iam')
#
#    user_groups = []
#    response = iam.list_groups_for_user(UserName=user)
#    for group in response['Groups']:
#        user_groups.append(group['GroupName'])
#    return user_groups
#
#
#def get_attached_policies(session, group):
#    iam = session.client('iam')
#
#    policies = []
#    response = iam.list_attached_group_policies(GroupName=group)
#    for policy in response['AttachedPolicies']:
#        policies.append(policy['PolicyName'])
#    return policies
#
#
#
#
#def assign_policies(session, group):
#    iam = session.client('iam')
#    flag = False
#    policies = get_attached_policies(session, group)
#    for policy in MANAGED_POLICIES:
#        if policy not in policies:
#            flag = True
#            break
#    if flag:
#        for policy in MANAGED_POLICIES:
#            arn = ARN_PREFIX + policy
#            iam.attach_group_policy(GroupName=group, PolicyArn=arn)
#
#
#
#
#def verify_creation(item):
#    print('\n + Dispatch recommends creating an IAM {} specific\n'
#          '   to KOPS administration to ensure principal of least privilege.\n'.format(str(item)))
#    create_item = input(' Create Dispatch KOPS admin {}?([y]/n) '.format(str(item))) or 'y'
#    return bool(create_item in ('y', 'Y', 'yes', 'Yes'))
#
#
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
    Use AWS session credentials to ensure dispatch
    has proper access to IAM and S3
    '''
    print('\n    Testing provided credentials...')
    iam_client = session.client('iam')
    s3_client = session.client('s3')
    try:
        iam_client.list_users()
        iam_client.list_groups()
        s3_client.list_buckets()
        print('    ...credentials successfully authenticated.\n')
    except botocore.exceptions.ClientError as creds_error:
        print("\n - There is an issue with the provided Access Key credentials:\n")
        print(creds_error)
        sys.exit(1)


def must_mount(access_key, user):
    '''
    Mounts /root directory of dispatch instance
    to dispatch user home directory
    '''
    if os.path.isdir("/root/.kube") or os.path.isdir("/root/.ssh"):
        print(' . Found container mount for /root.')
    else:
        mount_message = f'''
        You must mount /root to your home directory.
        Use the docker command below to properly operate KOPS:

        docker run --rm -it \\
        -e AWS_ACCESS_KEY_ID="{access_key}" \\
        -e USER="{user}" \\
        -v $HOME:/root \\
        registry.gitlab.com/christiantragesser/dispatch
        '''
        print(mount_message)
        sys.exit(1)


def kops_deps(session, name):
    '''
    Checks for existing KOPS S3 bucket,
    creates S3 bucket if doesn't exist
    '''
    s3_client = session.client('s3')

    kops_bucket = name+'-dispatch-kops-state-store'

    user_details = {}
    user_details['bucket'] = kops_bucket

    # Create KOPS S3 bucket
    buckets = get_s3_buckets(session)
    if kops_bucket in buckets:
        print(' . Using s3://{0:s} for KOPS state.'.format(kops_bucket))
    else:
        print(' + Creating KOPS state S3 bucket: {0:s}'.format(kops_bucket))
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

    return user_details
