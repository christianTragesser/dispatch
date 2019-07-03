#!/usr/bin/python
import boto3
import os
import sys

managedPolicies = ['AmazonEC2FullAccess', 'AmazonRoute53FullAccess', 'AmazonS3FullAccess', 'IAMFullAccess', 'AmazonVPCFullAccess']
arn_prefix = 'arn:aws:iam::aws:policy/'

def getUsers(session):
    iam = session.client('iam')
    
    users = []
    paginator = iam.get_paginator('list_users')
    for response in paginator.paginate():
        for user_name in response['Users']:
            users.append(user_name['UserName'])
    return users

def getGroups(session):
    iam = session.client('iam')

    groups = []
    paginator = iam.get_paginator('list_groups')
    for response in paginator.paginate():
        for group_name in response['Groups']:
            groups.append(group_name['GroupName'])
    return groups

def getUserGroups(session, user):
    iam = session.client('iam')

    userGroups = []
    response = iam.list_groups_for_user(UserName=user)
    for group in response['Groups']:
        userGroups.append(group['GroupName'])
    return userGroups

def getAttachedPolicies(session, group):
    iam = session.client('iam')

    policies = []
    response = iam.list_attached_group_policies(GroupName=group)
    for policy in response['AttachedPolicies']:
        policies.append(policy['PolicyName'])
    return policies

def getS3buckets(session):
    s3 = session.client('s3')

    nameList = []
    buckets = s3.list_buckets()
    for bucket in buckets['Buckets']:
        nameList.append(bucket['Name'])
    return nameList

def assignPolicies(session, group):
    iam = session.client('iam')
    flag = False
    policies = getAttachedPolicies(session, group)
    for policy in managedPolicies:
        if policy not in policies:
            flag = True
            break
    if flag:
        for policy in managedPolicies:
            arn = arn_prefix + policy
            iam.attach_group_policy(GroupName=group, PolicyArn=arn) 


def setCreds(access_key_id, secret_access_key, session_token):
    session = boto3.Session(
        aws_access_key_id=access_key_id,
        aws_secret_access_key=secret_access_key,
        aws_session_token=session_token
    )
    return session
    

def exerciseCreds(session):
    print('\n    Testing provided credentials...')
    iam = session.client('iam')
    s3 = session.client('s3')
    try:
        iam.list_users()
        iam.list_groups()
        s3.list_buckets()
        print('    ...credentials successfully authenticated.\n')
    except Exception as e:
        print(e)
        return sys.exit()

def verifyCreation(item):
    print('\n + Dispatch recommends creating an IAM {} specific\n'
          '   to KOPS administration to ensure principal of least privilege.\n'.format(str(item)))
    createItem = input(' Create Dispatch KOPS admin {}?([y]/n) '.format(str(item))) or 'y'
    if createItem == 'y' or createItem == 'Y' or createItem == 'yes' or createItem == 'Yes':
        return True
    else:
        return False

def kopsDeps(session, name, org):
    print('\n KOPS dependency checks:')
    s3 = session.client('s3')

    kopsBucket = name+'-dispatch-kops-state-store'

    userDetails = {}
    userDetails['bucket'] = kopsBucket

    #Create KOPS S3 bucket
    buckets = getS3buckets(session)
    if kopsBucket in buckets:
        print(' . Using s3://{0:s} for KOPS state.'.format(kopsBucket))
    else:
        print(' + Creating KOPS state S3 bucket: {0:s}'.format(kopsBucket))
        s3.create_bucket(ACL='private', Bucket=kopsBucket, )
        s3.put_bucket_encryption(
            Bucket=kopsBucket,
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
        s3.put_bucket_versioning(
                         Bucket=kopsBucket,
                         VersioningConfiguration={'Status': 'Enabled'}
        )

    return userDetails