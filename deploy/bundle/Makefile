ifndef IMAGE
IMAGE=quay.io/deployment-validation-operator/deployment-validation-operator-bundle
endif

ifndef IMAGE_TAG
IMAGE_TAG=latest
endif

MANIFEST_DIR=manifests
CSV=${MANIFEST_DIR}/deploymentvalidationoperator.clusterserviceversion.yaml

all: image

manifest:
	mkdir -p ${MANIFEST_DIR}
	oc process --local -o yaml --raw=true IMAGE=${IMAGE} IMAGE_TAG=${IMAGE_TAG} VERSION=${VERSION} REPLACE_VERSION=${REPLACE_VERSION} -f template/deploymentvalidationoperator.clusterserviceversion.yaml > ${CSV}

image: manifest
	docker build -t ${IMAGE}:${IMAGE_TAG} .

push: image
	docker push ${IMAGE}:${IMAGE_TAG}

clean:
	rm -rf ${MANIFEST_DIR}
