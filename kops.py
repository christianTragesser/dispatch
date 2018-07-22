import os, sys
from subprocess import call
import boto3
from botocore.exceptions import EndpointConnectionError

def giveMeShell(session, bucket):
  creds = session.get_credentials()
  creds = creds.get_frozen_credentials()
  os.environ['AWS_ACCESS_KEY_ID'] = creds.access_key
  os.environ['AWS_SECRET_ACCESS_KEY'] = creds.secret_key
  os.environ['KOPS_STATE_STORE'] = 's3://'+bucket
  os.system('/bin/sh')

def kopsSSHkey():
  sshKeyDir = '/root/.ssh'
  sshKey = sshKeyDir+'/kops_rsa'
  if not os.path.isfile(sshKey):
    print '\n* KOPS RSA key not found, generating...'
    print ' + Creating RSA key %s' % sshKey
    call(['ssh-keygen', '-t', 'rsa',
      '-b', '2048',
      '-q',
      '-N', '',
      '-f', sshKey
    ])

def describeAzs(session, region):
  ec2 = session.client('ec2', region_name=region)

  response = ec2.describe_availability_zones(
    Filters = [
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

def createOption(session, bucket):
  cluster_type = 'Quick'
  cluster_name = raw_input('\n New cluster FQDN(ex: test.k8s.local): ')
  region = raw_input(' Which AWS region?(ex: us-east-1): ')
  try:
    azs = describeAzs(session, region)
  except EndpointConnectionError:
    print " ! There was an issue with the AWS region you entered, let's try again."
    createOption(session, bucket)
  
  print '''
  New cluster details:
    Cluster name: %s
    Cluster type: %s
    AWS region: %s
  ''' % (cluster_name, cluster_type, region)
  verification = raw_input(' Create this cluster?(y/n) ')
  if verification == 'y' or verification == 'Y':
    try:
      kopsSSHkey()      
      createCluster(session, cluster_name, bucket, azs)
    except Exception as err:
      print err
      sys.exit(1)
  else:
    print 'exiting.'
    sys.exit(0)

def createCluster(session, name, bucket, azs):
  creds = session.get_credentials()
  creds = creds.get_frozen_credentials()
  os.environ['AWS_ACCESS_KEY_ID'] = creds.access_key
  os.environ['AWS_SECRET_ACCESS_KEY'] = creds.secret_key
  os.environ['KOPS_STATE_STORE'] = 's3://'+bucket

  labels = "owner=%s, cost_savings=True" % name
  print 'Creating cluster %s.k8s.local' % name
  print 'Using KOPS store @ s3://%s' % bucket

  call(['kops', 'create', 'cluster',
    '--zones='+azs[0],
    '--node-size=m4.large',
    '--topology=private',
    '--networking=weave',
    '--cloud-labels='+labels,
    '--name='+name,
    '--state=s3://'+bucket,
    '--ssh-public-key=~/.ssh/kops_rsa.pub',
    '--authorization=RBAC',
    '--yes'
  ])

def listKOPSclusters(session, bucket):
  s3 = session.client('s3')  
  response = s3.list_objects_v2(Bucket=bucket, Delimiter='/')
  if 'CommonPrefixes' in response:
    print ' Existing KOPS clusters:'
    for cluster in response['CommonPrefixes']:
      print '  - %s' % cluster['Prefix'].replace('/', '')
  else:
    print '   No clusters found.'

def deleteOption(session, bucket):
  listKOPSclusters(session, bucket)
  name = raw_input('\n FQDN of cluster to delete: ')
  verification = raw_input(' Are you SURE you want to delete cluster %s?(yes/no) ' % name)
  if verification == 'yes' or verification == 'Yes':
    try:
      deleteCluster(session, name, bucket)
    except Exception as err:
      print err
      sys.exit(1)
  else:
    print "\n You must type out 'yes' to confirm deletion. Let's try again...."
    deleteOption(session, bucket)

def deleteCluster(session, name, bucket):
  creds = session.get_credentials()
  creds = creds.get_frozen_credentials()
  os.environ['AWS_ACCESS_KEY_ID'] = creds.access_key
  os.environ['AWS_SECRET_ACCESS_KEY'] = creds.secret_key
  os.environ['KOPS_STATE_STORE'] = 's3://'+bucket

  call(['kops', 'delete', 'cluster',
    '--name='+name,
    '--state=s3://'+bucket,
    '--yes'
  ])