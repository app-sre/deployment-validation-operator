OPERATOR_NAME?=$(shell sed -n 's/.*OperatorName .*"\([^"]*\)".*/\1/p' config/config.go)
OPERATOR_NAMESPACE?=$(shell sed -n 's/.*OperatorNamespace .*"\([^"]*\)".*/\1/p' config/config.go)

ifndef IMAGE
IMAGE=quay.io/deployment-validation-operator/${OPERATOR_NAME}
endif

ifndef IMAGE_TAG
IMAGE_TAG=latest
endif


OUTDIR := _output
TESTTARGETS := $(shell go list -e ./... | egrep -v "/(vendor)/")
TESTOPTS :=

all: ${OUTDIR}/manager

${OUTDIR}/manager:
	GOARCH=amd64 go build -mod vendor -o ${OUTDIR}/manager cmd/manager/main.go

container: export GOOS=linux
container: clean ${OUTDIR}/manager
	docker build -t ${IMAGE}:${IMAGE_TAG} -f ./Dockerfile .

push: container
	docker push ${IMAGE}:${IMAGE_TAG}

${OUTDIR}:
	mkdir -p ${OUTDIR}

clean:
	rm -rf ${OUTDIR}

lint:
	GO111MODULE=on go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.31.0
	$(GOPATH)/bin/golangci-lint run ./.../


gotest:
	go test $(TESTOPTS) $(TESTTARGETS)
