#!/usr/bin/python
'''
dispatch main
'''
import os
import sys
from getpass import getpass
import utils
import kops

access_key_id = os.environ['AWS_ACCESS_KEY_ID'] if 'AWS_ACCESS_KEY_ID' in os.environ else None
# pylint: disable=C0301
secret_access_key = os.environ['AWS_SECRET_ACCESS_KEY'] if 'AWS_SECRET_ACCESS_KEY' in os.environ else None
session_token = os.environ['AWS_SESSION_TOKEN'] if 'AWS_SESSION_TOKEN' in os.environ else None
user_name = os.environ['USER'] if 'USER' in os.environ else None

WELCOME = '''
*********************************************************************
Thank you for using Dispatch. In the interest of security, only
temporary AWS credentials can be used to provision clusters.  See the
AWS documentation for creating temporary credentials for account IAM
users.
https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html
(Your IAM user needs appropriate access to provision all components)

Supplying your AWS access credentials and username as environment variables
will prevent this message from showing on Dispatch utilsiation:

docker run --rm -it \\
    -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \\
    -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \\
    -e AWS_SESSION_TOKEN=$AWS_SESSION_TOKEN \\
    -v $HOME/.dispatch:/root \\
    registry.gitlab.com/christiantragesser/dispatch

*********************************************************************
'''

# pylint: disable=W1401
ASCII_ART='''
 ______  _____ _______  _____  _______ _______ _______ _     _
 |     \   |   |______ |_____] |_____|    |    |       |_____|
 |_____/ __|__ ______| |       |     |    |    |______ |     |

'''
print(ASCII_ART)

try:
    if access_key_id is None and user_name is None:
        print(WELCOME)

    if access_key_id is None:
        print('***: KOPS credentials :***')
        access_key_id = input('Please enter your AWS Access Key ID: ')

    if secret_access_key is None or secret_access_key == '':
        secret_access_key = getpass('Please enter AWS Secret Access Key(masked input): ')

    if session_token is None or session_token == '':
        session_token = getpass('Please enter Session Token(masked input): ')

    kops_creds = utils.set_creds(access_key_id, secret_access_key, session_token)

    utils.exercise_creds(kops_creds)

    if user_name is None:
        user_name = input('Please enter your username: ')

    print('\n KOPS dependency checks:')
    utils.must_mount(access_key_id, user_name)
    user_detail = utils.kops_deps(kops_creds, user_name)

    kops.list_kops_clusters(kops_creds, user_detail['bucket'])

    print('''\nDispatch Menu:
      [C]reate new KOPS cluster
      [D]elete an existing KOPS cluster
      [Q]uit
      [*] Shell session
    ''')

    choice = {
      'C': kops.create_option,
      'c': kops.create_option,
      'D': kops.delete_option,
      'd': kops.delete_option
    }

    option = input(' Please select an [option]: ') or '*'

    if option in ('Q', 'q'):
        sys.exit(0)
    else:
        action = choice.get(option, kops.give_me_shell)
        action(kops_creds, user_detail['bucket'])
except KeyboardInterrupt:
    print("\n\n Keyboard interuption, we gone!")
