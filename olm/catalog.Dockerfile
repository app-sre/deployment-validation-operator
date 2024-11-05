# The builder image is expected to contain
# /bin/opm (with serve subcommand)
FROM quay.io/operator-framework/opm:latest

# Configure the entrypoint and command
ENTRYPOINT ["/bin/opm"]
CMD ["serve", "/configs", "--cache-dir=/tmp/cache"]

# Copy FBC root into image at /configs and pre-populate serve cache
ADD deployment-validation-operator-index /configs

# Set FBC-specific label for the location of the FBC root directory
# in the image
LABEL operators.operatorframework.io.index.configs.v1=/configs
