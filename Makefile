ifndef IMAGE
IMAGE=quay.io/deployment-validation-operator/dv-operator
endif

ifndef IMAGE_TAG
IMAGE_TAG=latest
endif

OUTDIR := _output

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
