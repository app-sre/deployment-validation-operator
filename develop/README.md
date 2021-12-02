# Local Development Environment

The scripts and manifests in this folder assist in creating a development environment for rapid iteration on `deployment-validation-operator`. Scripts for installing and bootstrapping the environment are included.

Features:

- Multi-node Kubernetes cluster in Docker with [K3D](https://k3d.io/)
- Local image registry server
- On golang / operator code changes, trigger automated container builds + Kubernetes workload updates with [Tilt](https://tilt.dev/)

## Usage

### Install & Configure

1. Run install

`$ bash install.sh`

2. Edit `./develop/registries.yaml`:

```
configs:
  quay.io:
    auth:
      username: ""
      password: ""
```

3. Capture system-specific build command

- Run `make` in deployment-validation-operator root directory
- Capture build command (last command seen after running `make`) and save to `devel/Dockerfile`

```
$ cd deployment-validation-operator && make
...
...
# Force GOOS=linux as we may want to build containers in other *nix-like systems (ie darwin).
# This is temporary until a better container build method is developed
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GOFLAGS=-mod=vendor GOOS=linux go build -gcflags="all=-trimpath=" -asmflags="all=-trimpath=" -o build/_output/bin/deployment-validation-operator ./cmd/manager

$ vim devel/Dockerfile

# diff
- RUN <make command output here>
+ RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GOFLAGS=-mod=vendor GOOS=linux go build -gcflags="all=-trimpath=" -asmflags="all=-trimpath=" -o build/_output/bin/deployment-validation-operator ./cmd/manager
```

### Bootstrap

`$ bash bootstrap.sh`

### Develop

`$ tilt up`

### Teardown

`$ k3d cluster delete dvo-local`

## Tooling

### K3D

[K3D](https://k3d.io/)

### Tilt

[Tilt](https://tilt.dev/)
