#!/usr/bin/python3
import os
from argparse import ArgumentParser
from pyplineCI import Pipeline

dirPath = os.path.dirname(os.path.realpath(__file__))
pipeline = Pipeline(dockerRegistry='registry.gitlab.com/christiantragesser')
localTag = 'local/dispatch'

def ci(option):
    stage = {
        'test': test,
        'local': local
    }
    run = stage.get(option, test)
    run()

def test():
    testDir = '/tmp/test'
    volumes = {
        dirPath: { 'bind': '/tmp', 'mode': 'rw'}
    }
    print('Starting tests:')
    pipeline.buildImage(dirPath, localTag)
    pipeline.runI(image=localTag, name='dispatch-test',
                  working_dir=testDir, volumes=volumes,
                  command=['/bin/sh', '-C', '/tmp/test/test_basic.sh'])
    print('Testing complete')

def local():
    volumes = {
        dirPath: { 'bind': '/tmp', 'mode': 'rw'}
    }
    print('Initializing locally built instance:')
    pipeline.buildImage(dirPath,localTag)
    pipeline.runI(image=localTag, name='dispatch-local',
                  working_dir='/tmp', volumes=volumes, command='/bin/sh')

def main():
    parser = ArgumentParser(prog='ci-py')
    parser.add_argument('stage', type=str, help='run pipeline stage; test, local')
    args = parser.parse_args()
    ci(args.stage)

if __name__ == '__main__':
    main()