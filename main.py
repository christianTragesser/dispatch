#!/usr/bin/python
import os
import sys
from getpass import getpass
import init
import kops

access_key_id = os.environ['AWS_ACCESS_KEY_ID'] if 'AWS_ACCESS_KEY_ID' in os.environ else None
secret_access_key = os.environ['AWS_SECRET_ACCESS_KEY'] if 'AWS_SECRET_ACCESS_KEY' in os.environ else None
session_token = os.environ['AWS_SESSION_TOKEN'] if 'AWS_SESSION_TOKEN' in os.environ else None
user_name = os.environ['USER'] if 'USER' in os.environ else None

welcome = '''
*********************************************************************
Thank you for using Dispatch. In the interest of security, only
temporary AWS credentials can be used to provision clusters.  See the
AWS documentation for creating temporary credentials for account IAM
users.
https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html
(Your IAM user needs appropriate access to provision all components)

Supplying your Access Key as environment variable 'AWS_ACCESS_KEY_ID'
and username as environment variable 'USER' will prevent this message
from showing on Dispatch initiation:

docker run --rm -it \\
-e AWS_ACCESS_KEY_ID="<access_key_id>" \\
-e USER="<username>" \\
-v $HOME:/root \\
registry.gitlab.com/christiantragesser/dispatch

*********************************************************************
'''

print('''
 ______  _____ _______  _____  _______ _______ _______ _     _
 |     \   |   |______ |_____] |_____|    |    |       |_____|
 |_____/ __|__ ______| |       |     |    |    |______ |     |

''')
try:
    if access_key_id is None and user_name is None:
        print(welcome)

    if access_key_id is None:
        print('***: KOPS inititialization :***')
        access_key_id = input('Please enter your AWS Access Key ID: ')

    if secret_access_key is None or secret_access_key == '':
        secret_access_key = getpass('Please enter AWS Secret Access Key(masked input): ')

    if session_token is None or session_token == '':
        session_token = getpass('Please enter Session Token(masked input): ')

    kopsCreds = init.setCreds(access_key_id, secret_access_key, session_token)

    try:
        init.exerciseCreds(kopsCreds)
    except Exception:
        print("\n - There is an issue with the provided Access Key credentials.\n")
        sys.exit(1)

    if user_name is None:
        user_name = input('Please enter your username: ')

    print('\n KOPS dependency checks:')
    init.mustMount(access_key_id, user_name)
    userDetail = init.kopsDeps(kopsCreds, user_name)

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
