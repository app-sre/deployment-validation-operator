ifndef IMAGE
IMAGE=quay.io/deployment-validation-operator/deployment-validation-operator-bundle
endif

ifndef IMAGE_TAG
IMAGE_TAG=latest
endif

ifndef OPERATOR_IMAGE
OPERATOR_IMAGE=quay.io/deployment-validation-operator/dv-operator
endif

ifndef OPERATOR_IMAGE_TAG
OPERATOR_IMAGE_TAG=latest
endif

MANIFEST_DIR=manifests
CSV=${MANIFEST_DIR}/deploymentvalidationoperator.clusterserviceversion.yaml

all: image

manifest:
	@mkdir -p $(MANIFEST_DIR) ; \
	TEMPLATE=`mktemp` ; \
	./generate-csv-template.py > $${TEMPLATE} ; \
	oc process --local -o yaml --raw=true IMAGE=$(OPERATOR_IMAGE) IMAGE_TAG=$(OPERATOR_IMAGE_TAG) VERSION=$(VERSION) REPLACE_VERSION=$(REPLACE_VERSION) -f $${TEMPLATE} > $(CSV)

bundle: manifest
	docker build -t ${IMAGE}:${IMAGE_TAG} .

push: bundle
	docker push ${IMAGE}:${IMAGE_TAG}

clean:
	rm -rf ${MANIFEST_DIR}
