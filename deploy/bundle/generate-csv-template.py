#!/usr/bin/env python3

import datetime
import os
import sys
import yaml
import pathlib
import argparse

parser = argparse.ArgumentParser()
parser.add_argument("-r", "--replaces", help="Replaces version", type=str)
parser.add_argument('-s','--skip', action='append',
        help='Skips version (can be specified multiple times)')
args = parser.parse_args()

root = pathlib.Path(__file__).parent.absolute() / '../..'
manifest_dir = root / 'deploy/openshift'
csv_template_dir = root / 'deploy/bundle/template'

with open(csv_template_dir / 'deploymentvalidationoperator.clusterserviceversion.yaml',
        'r') as stream:
    template = yaml.safe_load(stream)

csv = template['objects'][0]

csv['spec']['install']['spec']['clusterPermissions'] = []
with open(manifest_dir / 'cluster-role.yaml', 'r') as stream:
    operator_role = yaml.safe_load(stream)
    csv['spec']['install']['spec']['clusterPermissions'].append(
        {
            'rules': operator_role['rules'],
            'serviceAccountName': 'deployment-validation-operator',
        })

csv['spec']['install']['spec']['permissions'] = []
with open(manifest_dir / 'role.yaml', 'r') as stream:
    operator_role = yaml.safe_load(stream)
    csv['spec']['install']['spec']['permissions'].append(
        {
            'rules': operator_role['rules'],
            'serviceAccountName': 'deployment-validation-operator',
        })

with open(manifest_dir / 'operator.yaml', 'r') as stream:
    operator_components = []
    operator = yaml.safe_load_all(stream)
    for doc in operator:
        operator_components.append(doc)
    # There is only one yaml document in the operator deployment
    operator_deployment = operator_components[0]
    csv['spec']['install']['spec']['deployments'][0]['spec'] = \
        operator_deployment['spec']

csv['spec']['install']['spec']['deployments'][0]['spec']['template']['spec']['containers'][0]['image'] = \
    '${IMAGE}:${IMAGE_TAG}'

if args.replaces:
    csv['spec']['replaces'] = f'deployment-validation-operator.v{args.replaces}'

if args.skip:
    csv['metadata']['annotations']['olm.skipRange'] = ' || '.join(
        sorted(args.skip, key=lambda v: int(v.split('.')[2].split('-')[0])))

now = datetime.datetime.now()
csv['metadata']['annotations']['createdAt'] = \
    now.strftime('%Y-%m-%dT%H:%M:%SZ')

yaml.dump(template, sys.stdout, default_flow_style=False)
