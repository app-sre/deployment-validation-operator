#!/bin/bash -e

source `dirname $0`/common.sh

usage() { echo "Usage: $0 -o operator_name -c saas-repository-channel -H operator_commit_hash -n operator_commit_number [-p]" 1>&2; exit 1; }

while getopts "o:c:n:H:p" option; do
    case "${option}" in
        c)
            operator_channel=${OPTARG}
            ;;
        H)
            operator_commit_hash=${OPTARG}
            ;;
        n)
            operator_commit_number=${OPTARG}
            ;;
        o)
            operator_name=${OPTARG}
            ;;
        p)
            push_catalog=true
            ;;
        *)
            usage
    esac
done

# Checking parameters
check_mandatory_params operator_channel operator_name operator_commit_hash operator_commit_number

# Calculate previous version
SAAS_OPERATOR_DIR="saas-${operator_name}-bundle"
BUNDLE_DIR="${SAAS_OPERATOR_DIR}/${operator_name}"
OPERATOR_NEW_VERSION=$(ls "${BUNDLE_DIR}" | sort -t . -k 3 -g | tail -n 1)
OPERATOR_PREV_VERSION=$(ls "${BUNDLE_DIR}" | sort -t . -k 3 -g | tail -n 2 | head -n 1)

# Checking SAAS_OPERATOR_DIR exist
if [ ! -d "${SAAS_OPERATOR_DIR}/.git" ] ; then
    echo "${SAAS_OPERATOR_DIR} should exist and be a git repository"
    exit 1
fi

# create package yaml
cat <<EOF > $BUNDLE_DIR/${operator_name}.package.yaml
packageName: ${operator_name}
channels:
- name: ${operator_channel}
  currentCSV: ${operator_name}.v${OPERATOR_NEW_VERSION}
EOF

# add, commit & push
pushd ${SAAS_OPERATOR_DIR}

git add .

MESSAGE="add version ${operator_commit_number}-${operator_commit_hash}

replaces ${OPERATOR_PREV_VERSION}
removed versions: ${REMOVED_VERSIONS}"

git commit -m "${MESSAGE}"
git push origin "${operator_channel}"

if [ $? -ne 0 ] ; then 
    echo "git push failed, exiting..."
    exit 1
fi

popd

if [ "$push_catalog" = true ] ; then
    REGISTRY_IMG="quay.io/app-sre/${operator_name}-registry"
    
    # push image
    skopeo copy --dest-creds "${QUAY_USER}:${QUAY_TOKEN}" \
        "docker-daemon:${REGISTRY_IMG}:${operator_channel}-latest" \
        "docker://${REGISTRY_IMG}:${operator_channel}-latest"

    if [ $? -ne 0 ] ; then 
        echo "skopeo push of ${REGISTRY_IMG}:${operator_channel}-latest failed, exiting..."
        exit 1
    fi
    
    skopeo copy --dest-creds "${QUAY_USER}:${QUAY_TOKEN}" \
        "docker-daemon:${REGISTRY_IMG}:${operator_channel}-latest" \
        "docker://${REGISTRY_IMG}:${operator_channel}-${operator_commit_hash}"

    if [ $? -ne 0 ] ; then 
        echo "skopeo push of ${REGISTRY_IMG}:${operator_channel}-${operator_commit_hash} failed, exiting..."
        exit 1
    fi
fi