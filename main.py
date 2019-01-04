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
Thank you for using Dispatch. It is suggested to use a
Dispatch specific IAM user(dispatch-kops-admin-<user>) to ensure
KOPS automation operates as principle of least priviledge.

If you've already created a Dispatch admin user, supply the generated
Access Key as the environment variable 'AWS_ACCESS_KEY_ID'
on initiation of a Dispatch container instance:

docker run --rm -it \\
-e AWS_ACCESS_KEY_ID="<access_key_id>" \\
-e AWS_SECRET_ACCESS_KEY="" \\
-v $HOME:/root \\
registry.gitlab.com/christiantragesser/dispatch

******************************************************************
'''

print('''
 ______  _____ _______  _____  _______ _______ _______ _     _
 |     \   |   |______ |_____] |_____|    |    |       |_____|
 |_____/ __|__ ______| |       |     |    |    |______ |     |

''')
try:
  onboard = False
  if access_key_id is None:
    print(welcome)
    print('***: KOPS inititialization :***')
    onboard = True
    access_key_id = input('Please enter admin AWS Access Key ID: ')
  
  if secret_access_key is None or secret_access_key == '':
    secret_access_key = getpass('Please enter admin AWS Secret Access Key(masked input): ')
    
  kopsCreds = init.setCreds(access_key_id, secret_access_key)
  try:
    init.exerciseCreds(kopsCreds)
  except:
    print("\n - There is an issue with the provided Access Key credentials.\n")
    sys.exit(1)
  
  if user_name is None:
    user_name = input('Please enter your username: ')
  
  if org is None:
    org = input('Please enter your organization ID: ')
  
  userDetail = init.kopsDeps(kopsCreds, user_name, org)
  if onboard is True and userDetail['AccessKeyId'] is not None:
    print('''***: KOPS inititialization complete :***
  
      Use the docker command below to securely operate KOPS:
      (you recorded the Secret Access Key, right?)
  
      docker run --rm -it \\
      -e AWS_ACCESS_KEY_ID="{0:s}" \\
      -e AWS_SECRET_ACCESS_KEY="" \\
      -e NAME="{1:s}" \\
      -e ORG="{2:s}" \\
      -v $HOME:/root \\
      registry.gitlab.com/christiantragesser/dispatch
    '''.format(userDetail['AccessKeyId'], user_name, org))
    sys.exit(0)
  
  print('''\nDispatch Menu:
    [1] Create new KOPS cluster
    [2] List organization clusters
    [3] Delete an existing KOPS cluster
    [Q] Quit
    [*] Just give me a shell already!
  ''')
  
  choice = {
    '1': kops.createOption,
    '2': kops.listKOPSclusters,
    '3': kops.deleteOption
  }
  
  option = input(' Please select an [option]: ') or '*'
  if option == 'Q' or option == 'q':
    sys.exit(0)
  else:
    action = choice.get(option, kops.giveMeShell)
    action(kopsCreds, userDetail['bucket'])
except KeyboardInterrupt:
  print("\n\n Keyboard interuption, we gone!")