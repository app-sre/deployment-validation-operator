FROM registry.access.redhat.com/ubi9/go-toolset:1787774847 AS builder

USER root

RUN mkdir -p /workdir
COPY . /workdir
WORKDIR /workdir
ENV FIPS_ENABLED=1
RUN make go-build

FROM registry.access.redhat.com/ubi9/ubi-minimal:1786987606

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
      release="17"

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
