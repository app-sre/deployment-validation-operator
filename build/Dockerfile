FROM registry.access.redhat.com/ubi8/go-toolset:1.25-1769010432 AS builder

USER root

RUN mkdir -p /workdir
COPY . /workdir
WORKDIR /workdir
ENV FIPS_ENABLED=1
RUN make go-build

# This image has to match RHEL version from the CI or will cause some side-effects on glibc (and possibly others)
FROM registry.access.redhat.com/ubi8/ubi-minimal:8.10-1756195339

ENV OPERATOR=/usr/local/bin/deployment-validation-operator \
    USER_UID=1001 \
    USER_NAME=deployment-validation-operator

# install operator binary
COPY --from=builder /workdir/build/_output/bin/deployment-validation-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

# These labels are needed in order to release on Konflux
LABEL name="dvo/deployment-validation-rhel8-operator" \
      summary="Deployment Validation Operator for OpenShift" \
      description="Deployment Validation Operator for OpenShift" \
      com.redhat.component="deployment-validation-operator-container" \
      io.k8s.display-name="Deployment Validation Operator" \
      io.k8s.description="Deployment Validation Operator for OpenShift" \
      io.openshift.tags="dvo,deployment-validation-operator" \
      cpe="cpe:/a:redhat:deployment_validator_operator:0.7::el8" \
      version="0.7" \
      release="14"

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
