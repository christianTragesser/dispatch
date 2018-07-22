#!/usr/bin/python
import os
import sys
from getpass import getpass
import init
import kops

access_key_id = os.environ['AWS_ACCESS_KEY_ID'] if 'AWS_ACCESS_KEY_ID' in os.environ else None
secret_access_key = os.environ['AWS_SECRET_ACCESS_KEY'] if 'AWS_SECRET_ACCESS_KEY' in os.environ else None
org = os.environ['ORG'] if 'ORG' in os.environ else None
user_name = os.environ['NAME'] if 'NAME' in os.environ else None

welcome = ''' 
******************************************************************
Thank you for using Dispatch. To begin, we'll need to create
your KOPS administration IAM user and Access Keys.  It is
suggested to use the newly created IAM user(kops-admin-<user>)
credentials to ensure KOPS automation operates as principle of
least priviledge.

If you've already created a KOPS admin user, supply the generated
Access Keys as environment variables 'AWS_ACCESS_KEY_ID' and
'AWS_SECRET_ACCESS_KEY' on start of a Dispatch container instance:

docker run --rm -it \\
-e AWS_ACCESS_KEY_ID="<access_key_id>" \\
-e AWS_SECRET_ACCESS_KEY="<secret_access_key>" \\
-v $HOME/root \\
christiantragesser/dispatch

******************************************************************
'''

print '''
 ______  _____ _______  _____  _______ _______ _______ _     _
 |     \   |   |______ |_____] |_____|    |    |       |_____|
 |_____/ __|__ ______| |       |     |    |    |______ |     |

'''
try:
  onboard = False
  if access_key_id is None:
    print welcome
    print '***: KOPS inititialization :***'
    onboard = True
    access_key_id = raw_input('Please enter admin AWS Access Key ID: ')
  
  if secret_access_key is None or secret_access_key == '':
    secret_access_key = getpass('Please enter admin AWS Secret Access Key(masked input): ')
    
  kopsCreds = init.setCreds(access_key_id, secret_access_key)
  try:
    init.exerciseCreds(kopsCreds)
  except:
    print "\n - There is an issue with the provided Access Key credentials.\n"
    sys.exit(1)
  
  if user_name is None:
    user_name = raw_input('Please enter your username: ')
  
  if org is None:
    org = raw_input('Please enter your organization ID: ')
  
  userDetail = init.kopsDeps(kopsCreds, user_name, org)
  
  if onboard is True and userDetail['AccessKeyId'] is not None:
    print '''***: KOPS inititialization complete :***
  
      Use the docker command below to securely operate KOPS:
      (you recorded the Secret Access Key, right?)
  
      docker run --rm -it \\
      -e AWS_ACCESS_KEY_ID="%s" \\
      -e AWS_SECRET_ACCESS_KEY="" \\
      -e NAME="%s" \\
      -e ORG="%s" \\
      -v $HOME:/root \\
      christiantragesser/dispatch
    ''' % (userDetail['AccessKeyId'], user_name, org)
    sys.exit(0)
  
  print'''Dispatch Menu:
    [1] Create new KOPS cluster
    [2] List organization clusters
    [3] Delete an existing KOPS cluster
    [*] Just give me a shell already!
  '''
  
  choice = {
    '1': kops.createOption,
    '2': kops.listKOPSclusters,
    '3': kops.deleteOption
  }
  
  option = raw_input(' Please select an [option]: ')
  action = choice.get(option, kops.giveMeShell)
  action(kopsCreds, userDetail['bucket'])
except KeyboardInterrupt:
  print "\n Keyboard interuption, we gone!"