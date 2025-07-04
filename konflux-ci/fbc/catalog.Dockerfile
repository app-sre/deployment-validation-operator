ARG OCP_V=latest
ARG CAT_TYPE=bundle.object

# The builder image is expected to contain
# /bin/opm (with serve subcommand)
FROM registry.redhat.io/openshift4/ose-operator-registry-rhel9:$OCP_V as builder

# Copy FBC root into image at /configs and pre-populate serve cache
ARG CAT_TYPE
ADD $CAT_TYPE/catalog /configs
RUN ["/bin/opm", "serve", "/configs", "--cache-dir=/tmp/cache", "--cache-only"]

ARG OCP_V
# The base image is expected to contain
# /bin/opm (with serve subcommand) and /bin/grpc_health_probe
FROM registry.redhat.io/openshift4/ose-operator-registry-rhel9:$OCP_V

# Configure the entrypoint and command
ENTRYPOINT ["/bin/opm"]
CMD ["serve", "/configs", "--cache-dir=/tmp/cache"]

COPY --from=builder /configs /configs
COPY --from=builder /tmp/cache /tmp/cache

# Set FBC-specific label for the location of the FBC root directory
# in the image
LABEL operators.operatorframework.io.index.configs.v1=/configs
