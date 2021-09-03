'''
dispatch KOPS utilities
'''
# pylint: disable=R1732
# pylint: disable=W0107
# pylint: disable=R0913
import os
import sys
from subprocess import Popen
import requests
from botocore.exceptions import EndpointConnectionError

latest_kops_release = requests.get('https://api.github.com/repos/kubernetes/kops/releases/latest')
kops_version = latest_kops_release.json()['tag_name']
k8s_version = os.environ['K8S_VERSION'] if 'K8S_VERSION' in os.environ else kops_version[1:]


class KopsException(Exception):
    '''
    Custom exception for failed KOPS commands
    '''
    pass


def get_creds(session):
    '''
    set AWS creds for KOPS activities
    '''
    session_creds = session.get_credentials()
    frozen_creds = session_creds.get_frozen_credentials()

    return frozen_creds


def set_env_vars(credentials, bucket):
    '''
    assign dispatch session env vars
    '''
    os.environ['AWS_ACCESS_KEY_ID'] = credentials.access_key
    os.environ['AWS_SECRET_ACCESS_KEY'] = credentials.secret_key
    os.environ['AWS_SESSION_TOKEN'] = credentials.token
    os.environ['KOPS_STATE_STORE'] = 's3://'+bucket
    os.environ['KUBECONFIG'] = '/root/.dispatch/.kube/config'


def give_me_shell(session, bucket):
    '''
    drop user into container shell
    '''
    creds = get_creds(session)
    set_env_vars(creds, bucket)
    os.system('/bin/sh')


def describe_azs(session, region):
    '''
    returns available availability zones in region
    '''
    ec2_client = session.client('ec2', region_name=region)
    response = ec2_client.describe_availability_zones(
        Filters=[
          {
            'Name': 'region-name',
            'Values': [
              region
            ]
          }
        ]
      )

    zones = [x['ZoneName'] for x in response['AvailabilityZones']]

    return zones


def set_cluster_size():
    '''
    set kubernetes node EC2 instance size
    '''
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
    '''
    sets number of kubernetes nodes
    '''
    option = input(' Compute node count(2): ') or 2

    try:
        count = int(option)
    except ValueError:
        print(f"    '{option}' is not valid. Please provide an integer for node count.")
        get_node_count()

    return count


def create_cluster(session, name, bucket, azs, node_size, node_count):
    '''
    creates KOPS cluser
    '''
    creds = get_creds(session)
    labels = f'owner={name}, CreatedBy=Dispatch'
    kops_command = ['kops', 'create', 'cluster', '--zones='+azs[0],
                    '--node-size='+node_size['instance_size'],
                    '--node-count='+str(node_count),
                    '--topology=private',
                    '--kubernetes-version='+k8s_version,
                    '--networking=weave',
                    '--cloud-labels='+labels,
                    '--name='+name,
                    '--state=s3://'+bucket,
                    '--ssh-public-key=~/.dispatch/.ssh/kops_rsa.pub',
                    '--authorization=RBAC',
                    '--yes']

    print(f'\nUsing KOPS store @ s3://{bucket}')
    print(f'Creating cluster {name}:\n')

    set_env_vars(creds, bucket)
    process = Popen(kops_command)
    process.wait()

    if process.returncode != 0:
        raise KopsException(f'Provisioning of KOPS cluster {name} failed.\n')


def create_option(session, bucket):
    '''
    validates cluster options before creating cluster
    '''
    cluster_name = input('\n New cluster FQDN(dispatch.k8s.local): ') or 'dispatch.k8s.local'
    region = input(' AWS region(us-east-1): ') or 'us-east-1'
    try:
        azs = describe_azs(session, region)
    except EndpointConnectionError:
        print(" ! There was an issue with the AWS region you entered, let's try again.")
        create_option(session, bucket)

    node_size = set_cluster_size()
    node_count = get_node_count()

    print(f'''
    New cluster details
    -------------------
      Kubernetes version: {k8s_version}
      Cluster name: {cluster_name}
      Cluster size: {node_size['label']}
      Node count: {node_count}
      AWS region: {region}
    ''')
    verification = input(' Create this cluster?(y/n): ') or 'n'
    if verification in ('y', 'Y'):
        try:
            create_cluster(session, cluster_name, bucket, azs, node_size, node_count)
        except KopsException as kops_err:
            print(kops_err)
            sys.exit(1)
    else:
        print('exiting.')
        sys.exit(0)


def list_kops_clusters(session, bucket):
    '''
    list existing KOPS clusters found in S3 bucket store
    '''
    s3_client = session.client('s3')
    response = s3_client.list_objects_v2(Bucket=bucket, Delimiter='/')

    if 'CommonPrefixes' in response:
        print('\n Existing KOPS clusters:')
        for cluster in response['CommonPrefixes']:
            print('  - {0:s}'.format(cluster['Prefix'].replace('/', '')))
    else:
        print('\n Cluster list is empty.')


def delete_cluster(session, name, bucket):
    '''
    deletes existing clusters
    '''
    creds = get_creds(session)
    set_env_vars(creds, bucket)
    process = Popen(['kops', 'delete', 'cluster',
          '--name='+name,
          '--state=s3://'+bucket,
          '--yes'
          ])
    process.wait()

    if process.returncode != 0:
        raise KopsException(f'Deletion of KOPS cluster {name} failed.\n')


def delete_option(session, bucket):
    '''
    get options for cluster deletion
    '''
    list_kops_clusters(session, bucket)
    name = input('\n FQDN of cluster to delete: ') or ''
    print(f'\n Are you SURE you want to delete cluster {name}?')
    verification = input(' You must type "yes" to verify: ') or 'no'
    if verification in ('yes', 'Yes'):
        try:
            delete_cluster(session, name, bucket)
        except KopsException as delete_err:
            print(delete_err)
            sys.exit(1)
    else:
        print("\n You must type out 'yes' to confirm deletion, exiting.")
        sys.exit(0)
