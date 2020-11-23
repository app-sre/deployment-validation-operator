#!/usr/bin/env python3

# vim: ts=4:sw=4:cc=99

import datetime
import os
import sys
import yaml
import pathlib
import argparse

parser = argparse.ArgumentParser(add_help=False)
required = parser.add_argument_group('required arguments')

required.add_argument('-n', '--name', help='operator name', type=str, required=True)
required.add_argument('-c', '--current-version', help='operator version', type=str, required=True)
required.add_argument('-i', '--image', help='operator image', type=str, required=True)
required.add_argument('-t', '--image-tag', help='operator image tag', type=str, required=True)
required.add_argument('-o', '--output-dir', help='output directory', type=str, required=True)

optional = parser.add_argument_group('optional arguments')
optional.add_argument( '-h', '--help', action='help', default=argparse.SUPPRESS,
    help='show this help message and exit')
optional.add_argument('-r', '--replaces', help='Replaces version', type=str)
optional.add_argument('-s','--skip', action='append',
        help='Skips version (can be specified multiple times)')

args = parser.parse_args()

root = pathlib.Path(__file__).parent.absolute() / '..'
manifest_dir = root / 'deploy/openshift'
csv_template_dir = root / 'deploy/bundle/template'

with open(csv_template_dir / f'{args.name}.clusterserviceversion.yaml',
        'r') as stream:
    csv = yaml.safe_load(stream)

csv['spec']['install']['spec']['clusterPermissions'] = []
with open(manifest_dir / 'cluster-role.yaml', 'r') as stream:
    operator_role = yaml.safe_load(stream)
    csv['spec']['install']['spec']['clusterPermissions'].append(
        {
            'rules': operator_role['rules'],
            'serviceAccountName': args.name,
        })

csv['spec']['install']['spec']['permissions'] = []
with open(manifest_dir / 'role.yaml', 'r') as stream:
    operator_role = yaml.safe_load(stream)
    csv['spec']['install']['spec']['permissions'].append(
        {
            'rules': operator_role['rules'],
            'serviceAccountName': args.name,
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
    csv['metadata']['annotations']['containerImage'] = f'{args.image}:{args.image_tag}'
csv['metadata']['name'] = f'{args.name}.v{args.current_version}'
csv['spec']['version'] = args.current_version
csv['spec']['links'][1]['url'] = f'https://{args.image}:{args.image_tag}'

if args.replaces:
    csv['spec']['replaces'] = f'{args.name}.v{args.replaces}'

if args.skip:
    csv['metadata']['annotations']['olm.skipRange'] = ' || '.join(
        sorted(args.skip, key=lambda v: int(v.split('.')[2].split('-')[0])))

now = datetime.datetime.now()
csv['metadata']['annotations']['createdAt'] = \
    now.strftime('%Y-%m-%dT%H:%M:%SZ')

csv_filename = pathlib.Path(args.output_dir) / \
    f'{args.name}.v{args.current_version}.clusterserviceversion.yaml'
with open(csv_filename, 'w') as output_file:
    yaml.dump(csv, output_file, default_flow_style=False)
