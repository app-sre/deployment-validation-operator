# Deployment Validation Operator

## Description

The Deployment Validation Operator (DVO) checks deployments and other resources against a curated collection of best practices.

These best practices focus mainly on ensuring that the applications are fault-tolerant.

DVO will only monitor Kubernetes resources and will not modify them in any way. Instead, it will report failed validations via Prometheus, which will allow users of this operator to create alerts based on its results. All the metrics are gauges that will report `1` if the best-practice has failed. The metric will always have three parameters: `name`, `namespace` and `kind`.

This operator doesn't define any CRDs at the moment. It has been bootstrapped with `operator-sdk` making it possible to add a CRD in the future if required.

## Deployment

### Manual installation

There are manifests to install the operator under the [`deploy/openshift`](deploy/openshift) directory. A typical installation would go as follows:

* Create the `deployment-validation-operator` namespace/project
* Create the service, service account, configmap, roles and role bindings
* Create the operator deployment

```
oc new-project deployment-validation-operator
for manifest in service-account.yaml \
                service.yaml \
                role.yaml \
                cluster-role.yaml \
                role-binding.yaml \
                cluster-role-binding.yaml \
                configmap.yaml \
                operator.yaml
do
    oc create -f deploy/openshift/$manifest
done
```

### Install dashboard

There are manifests to install a simple grafana dashboard under the [`deploy/observability`](deploy/observability) directory. A typical installation would go as follows:

```
for manifest in deploy/observability/*
do
    oc create -f $manifest
done
```

## Tests

You can run the unit tests via

```
make test
```

We use [openshift boilerplate](https://github.com/openshift/boilerplate) to manage our make targets. See this [doc](https://github.com/openshift/boilerplate/blob/master/boilerplate/openshift/golang-osd-operator/README.md) for further information.

## Roadmap

- e2e tests

