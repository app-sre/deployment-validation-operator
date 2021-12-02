FROM quay.io/app-sre/boilerplate:image-v1.0.0 AS builder

RUN mkdir -p /workdir
COPY . /workdir
WORKDIR /workdir
RUN <make command output here>

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/deployment-validation-operator \
    USER_UID=1001 \
    USER_NAME=deployment-validation-operator

# install operator binary
COPY --from=builder /workdir/build/_output/bin/deployment-validation-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
