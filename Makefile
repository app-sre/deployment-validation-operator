OUTDIR := _output

all: ${OUTDIR}/manager

${OUTDIR}/manager:
	GOARCH=amd64 go build -o ${OUTDIR}/manager cmd/manager/main.go

container: export GOOS=linux
container: clean ${OUTDIR}/manager
	docker build -t stage.quay.io/stage-app-sre/dv-operator:latest -f ./Dockerfile .

${OUTDIR}:
	mkdir -p ${OUTDIR}

clean:
	rm -rf ${OUTDIR}
