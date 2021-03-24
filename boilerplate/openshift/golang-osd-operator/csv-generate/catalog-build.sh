#!/bin/bash -e

source `dirname $0`/common.sh

usage() { echo "Usage: $0 -o operator_name -c saas-repository-channel" 1>&2; exit 1; }

while getopts "o:H:c:" option; do
    case "${option}" in
        o)
            operator_name=${OPTARG}
            ;;
        c)
            operator_channel=${OPTARG}
            ;;
        *)
            usage
    esac
done

# Checking parameters
check_mandatory_params operator_channel operator_name  

# Parameters for the Dockerfile
SAAS_OPERATOR_DIR="saas-${operator_name}-bundle"
REGISTRY_IMG="quay.io/app-sre/${operator_name}-registry"
DOCKERFILE_REGISTRY="Dockerfile.olm-registry"

# Checking SAAS_OPERATOR_DIR exist
if [ ! -d "${SAAS_OPERATOR_DIR}/.git" ] ; then
    echo "${SAAS_OPERATOR_DIR} should exist and be a git repository"
    exit 1
fi

cat <<EOF > $DOCKERFILE_REGISTRY
FROM quay.io/openshift/origin-operator-registry:latest
COPY $SAAS_OPERATOR_DIR manifests
RUN initializer --permissive
CMD ["registry-server", "-t", "/tmp/terminate.log"]
EOF

docker build -f $DOCKERFILE_REGISTRY --tag "${REGISTRY_IMG}:${operator_channel}-latest" .

if [ $? -ne 0 ] ; then 
    echo "docker build failed, exiting..."
    exit 1
fi

# TODO : Test the image and the version it contains