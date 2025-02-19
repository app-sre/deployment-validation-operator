FROM registry.access.redhat.com/ubi9/go-toolset:1.22.9 AS builder

USER root

RUN mkdir -p /workdir
COPY . /workdir
WORKDIR /workdir
ENV FIPS_ENABLED=1
RUN make go-build

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.5-1738816775

# update glibc libraries in case it is needed
RUN microdnf update -y && \
    microdnf install glibc-static -y

ENV OPERATOR=/usr/local/bin/deployment-validation-operator \
    USER_UID=1001 \
    USER_NAME=deployment-validation-operator

# install operator binary
COPY --from=builder /workdir/build/_output/bin/deployment-validation-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

# These labels are needed in order to release on Konflux
LABEL name="deployment-validation-operator" \
      summary="Deployment Validation Operator for OpenShift" \
      description="Deployment Validation Operator for OpenShift" \
      com.redhat.component="deployment-validation-operator-container" \
      io.k8s.display-name="Deployment Validation Operator" \
      io.k8s.description="Deployment Validation Operator for OpenShift" \
      io.openshift.tags="dvo,deployment-validation-operator"

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
