FROM scratch

COPY ./manifests /manifests
COPY ./metadata /metadata

LABEL com.redhat.component="deployment-validation-operator-bundle-container" \
      name="app-sre/deployment-validation-operator-bundle" \
      version="0.7" \
      release="11" \
      distribution-scope="private" \
      vendor="Red Hat, Inc." \
      url="github.com/app-sre/deployment-validation-operator/" \
      summary="Deployment Validation Operator for OpenShift" \
      io.openshift.expose-services="" \
      io.k8s.display-name="Deployment Validation Operator Bundle" \
      io.k8s.description="Deployment Validation Operator Bundle for OpenShift" \
      maintainer="['dvo-owners@redhat.com']" \
      description="Deployment Validation Operator for OpenShift" \
      com.redhat.delivery.operator.bundle=true \
      com.redhat.openshift.versions="v4.14-v4.18" \
      operators.operatorframework.io.bundle.mediatype.v1=registry+v1 \
      operators.operatorframework.io.bundle.manifests.v1=manifests/ \
      operators.operatorframework.io.bundle.metadata.v1=metadata/ \
      operators.operatorframework.io.bundle.package.v1=deployment-validation-operator \
      operators.operatorframework.io.bundle.channels.v1=alpha \
      operators.operatorframework.io.metrics.builder=operator-sdk-v1.31.0+git \
      operators.operatorframework.io.metrics.mediatype.v1=metrics+v1 \
      operators.operatorframework.io.metrics.project_layout=unknown
