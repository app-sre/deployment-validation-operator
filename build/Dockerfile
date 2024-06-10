FROM quay.io/app-sre/boilerplate:image-v5.0.0 AS builder

RUN mkdir -p /workdir
COPY . /workdir
WORKDIR /workdir
RUN make

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.10-896.1717584414

ENV OPERATOR=/usr/local/bin/deployment-validation-operator \
    USER_UID=1001 \
    USER_NAME=deployment-validation-operator

# install operator binary
COPY --from=builder /workdir/build/_output/bin/deployment-validation-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
