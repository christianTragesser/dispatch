import os
import sys
from subprocess import call
from botocore.exceptions import EndpointConnectionError

k8s_version = '1.14.8'


def getCreds(session):
    session_creds = session.get_credentials()
    frozen_creds = session_creds.get_frozen_credentials()
    return frozen_creds


def setEnvVars(credentials, bucket):
    os.environ['AWS_ACCESS_KEY_ID'] = credentials.access_key
    os.environ['AWS_SECRET_ACCESS_KEY'] = credentials.secret_key
    os.environ['AWS_SESSION_TOKEN'] = credentials.token
    os.environ['KOPS_STATE_STORE'] = 's3://'+bucket


def giveMeShell(session, bucket):
    creds = getCreds(session)
    setEnvVars(creds, bucket)
    os.system('/bin/sh')


def kopsSSHkey():
    sshKeyDir = os.environ['HOME']+'/.ssh'
    sshKey = sshKeyDir+'/kops_rsa'
    if not os.path.isfile(sshKey):
        print('\n* KOPS RSA key not found, generating...')
        print(' + Creating RSA key {0:s}'.format(sshKey))
        call(['ssh-keygen', '-t', 'rsa',
              '-b', '2048',
              '-q',
              '-N', '',
              '-f', sshKey
              ])


def describeAzs(session, region):
    ec2 = session.client('ec2', region_name=region)
    response = ec2.describe_availability_zones(
        Filters=[
          {
            'Name': 'region-name',
            'Values': [
              region
            ]
          }
        ]
      )
    zones = []
    for az in response['AvailabilityZones']:
        zones.append(az['ZoneName'])
    return zones


def setClusterSize():
    print(''' Cluster Size:
      [S]mall - 2 CPU/4 GB RAM nodes
      [M]edium - 4 CPU/16 GB RAM nodes
      [L]arge - 8 CPU/32 GB RAM nodes''')

    option = input('    Size(s): ') or 's'
    sizes = {
      's': {'label': 'Small', 'instance_size': 't2.medium'},
      'S': {'label': 'Small', 'instance_size': 't2.medium'},
      'm': {'label': 'Medium', 'instance_size': 't2.xlarge'},
      'M': {'label': 'Medium', 'instance_size': 't2.xlarge'},
      'l': {'label': 'Large', 'instance_size': 'm4.2xlarge'},
      'L': {'label': 'Large', 'instance_size': 'm4.2xlarge'}
    }
    size = sizes.get(option, {'label': 'Small', 'instance_size': 't2.medium'})
    return size


def get_node_count():
    option = input(' Compute node count(2): ') or '2'
    try:
        return int(option)
    except ValueError:
        print("    '{}' is not a valid count. Please provide an integer value for node count.".format(option))
        get_node_count()


def createOption(session, bucket):
    cluster_name = input('\n New cluster FQDN(dispatch.k8s.local): ') or 'dispatch.k8s.local'
    region = input(' AWS region(us-east-1): ') or 'us-east-1'
    try:
        azs = describeAzs(session, region)
    except EndpointConnectionError:
        print(" ! There was an issue with the AWS region you entered, let's try again.")
        createOption(session, bucket)

    node_size = setClusterSize()
    node_count = get_node_count()

    print('''
    New cluster details:
      Cluster name: {0:s}
      Cluster size: {1:s}
      Node count: {2:d}
      AWS region: {3:s}
    '''.format(cluster_name, node_size['label'], node_count, region))
    verification = input(' Create this cluster?(y/n): ') or 'n'
    if verification == 'y' or verification == 'Y':
        try:
            kopsSSHkey()
            print('Attempting createCluster function')
            createCluster(session, cluster_name, bucket, azs, node_size, node_count)
        except Exception as err:
            print(err)
            sys.exit(1)
    else:
        print('exiting.')
        sys.exit(0)


def createCluster(session, name, bucket, azs, node_size, node_count):
    creds = getCreds(session)
    setEnvVars(creds, bucket)
    labels = 'owner={0:s}, CreatedBy=Dispatch'.format(name)
    print('Creating cluster {0:s}'.format(name))
    print('Using KOPS store @ s3://{0:s} \n'.format(bucket))
    kops_command = ['kops', 'create', 'cluster', '--zones='+azs[0],
                    '--node-size='+node_size['instance_size'],
                    '--node-count='+str(node_count),
                    '--topology=private',
                    '--kubernetes-version='+k8s_version,
                    '--networking=weave',
                    '--cloud-labels='+labels,
                    '--name='+name,
                    '--state=s3://'+bucket,
                    '--ssh-public-key=~/.ssh/kops_rsa.pub',
                    '--authorization=RBAC',
                    '--yes',
                    '--bastion']
    # if using gossip protocol domain, do not provision a bastion host
    if '.k8s.local' in name:
        del kops_command[-1]
    call(kops_command)


def listKOPSclusters(session, bucket):
    s3 = session.client('s3')
    response = s3.list_objects_v2(Bucket=bucket, Delimiter='/')
    if 'CommonPrefixes' in response:
        print('\n Existing KOPS clusters:')
        for cluster in response['CommonPrefixes']:
            print('  - {0:s}'.format(cluster['Prefix'].replace('/', '')))
    else:
        print('\n Cluster list is empty.')


def deleteOption(session, bucket):
    listKOPSclusters(session, bucket)
    name = input('\n FQDN of cluster to delete: ') or ''
    print('\n Are you SURE you want to delete cluster {0:s}?'.format(name))
    verification = input(' You must type "yes" to verify: ') or 'no'
    if verification == 'yes' or verification == 'Yes':
        try:
            deleteCluster(session, name, bucket)
        except Exception as err:
            print(err)
            sys.exit(1)
    else:
        print("\n You must type out 'yes' to confirm deletion. Let's try again....")
        deleteOption(session, bucket)


def deleteCluster(session, name, bucket):
    creds = getCreds(session)
    setEnvVars(creds, bucket)
    call(['kops', 'delete', 'cluster',
          '--name='+name,
          '--state=s3://'+bucket,
          '--yes'
          ])
