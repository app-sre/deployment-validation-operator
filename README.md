# Deployment Validation Operator

## Description

The Deployment Validation Operator (DVO) checks deployments and other resources against a curated collection of best practices. 

These best practices focus mainly on ensuring that the applications are fault-tolerant.

DVO will only monitor Kubernetes resources and will not modify them in any way. As an operator it is a continuously running version of the static analysis tool Kube-linter [https://github.com/stackrox/kube-linter]. It will report failed validations via Prometheus, which will allow users of this operator to create alerts based on its results. All the metrics are gauges that will report `1` if the best-practice has failed. The metric will always have three parameters: `name`, `namespace` and `kind`. 

This operator doesn't define any CRDs at the moment. It has been bootstrapped with `operator-sdk` making it possible to add a CRD in the future if required.

## Architecture Diagrams

[Architecure Diagrams](./docs/architecture.md)

## Deployment

The manifests to deploy DVO take a permissive approach to permissions.  This is done to make it easier to support monitoring new object kinds without having to change rbac rules.  This means that elevated permissions will be required in order to deploy DVO through standard manifests.  There is a manifest to deploy DVO though OLM from opereatorhub which does alleviate this need to have elevated permissions.

* DVO deployment should only deploy 1 pod as currently metrics are not replicated across a standard 3 causing installation issues (will be fixed in a later version)

### Manual installation (without OLM)

There are manifests to install the operator under the [`deploy/openshift`](deploy/openshift) directory. A typical installation would go as follows:

* Create the `deployment-validation-operator` namespace/project
    * If deploying to a namespace other than `deployment-validation-operator`, there are commented lines you must change in `deploy/openshift/cluster-role-binding.yaml` and `deploy/openshift/role-binding.yaml` first
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

### Installation via OLM

There is a manifest to deploy DVO via OLM artifacts.  This assumes that OLM is already running in the cluster.  To deploy via OLM:

* Generate the deployment YAML from the openshift template
* Deploy one of the two following YAML templates (not both!):

```
# deploy this if you DO NOT want OLM to automatically upgrade DVO
oc new-project deployment-validation-operator
oc process --local NAMESPACE_IGNORE_PATTERN='openshift.*|kube-.+' -f deploy/openshift/deployment-validation-operator-olm.yaml | oc apply -f -
```

```
# otherwise, deploy this if you DO want OLM to automatically upgrade DVO
# set REGISTRY_POLLING_INTERVAL to be shorter to have OLM check for new DVO versions more frequently if desired; e.g. '45m'
# the shorter the interval, the more resources OLM may consume
# read more about OLM catalog polling: https://github.com/operator-framework/
operator-lifecycle-manager/blob/master/doc/design/catalog-polling.md

oc new-project deployment-validation-operator
oc process --local \
NAMESPACE_IGNORE_PATTERN='openshift.*|kube-.+' \
REGISTRY_POLLING_INTERVAL='24h' \
-f deploy/openshift/deployment-validation-operator-olm-with-polling.yaml \
| oc apply -f -
```

If DVO is deployed to a namespace other than the one where OLM is deployed, which is usually the case, then a network policy may be required to allow OLM to see the artifacts in the DVO namespace.  For example, if OLM is deployed in the namespace `operator-lifecycle-manager` then the network policy would be deployed like this:

```
oc process --local NAMESPACE='operator-lifecycle-manager' -f deploy/openshift/network-policies.yaml | oc apply -f -
```

> Note: When installing on OSD it would be beneficial to use an expanced NAMESPACE_IGNORE_PATTERN like
`NAMESPACE_IGNORE_PATTERN='^(openshift.|kube-.|default|dedicated-admin|redhat-.|acm|addon-dba-operator|codeready-.|prow)$'`

## Install Grafana dashboard

There are manifests to install a simple grafana dashboard under the [`deploy/observability`](deploy/observability) directory.

A typical installation to the default namespace `deployment-validation-operator` goes as follows:
`oc process -f deploy/observability/template.yaml | oc create -f -`

Or, if you want to deploy deployment-validation-operator components to a custom namespace:
`oc process --local NAMESPACE="custom-dvo-namespace" -f deploy/observability/template.yaml | oc create -f -`

## Allow scraping from outside DVO namespace

The metrics generated by DVO can be scraped by anything that understands prometheus metrics.  A network policy may be needed to allow the DVO metrics to be collected from a service running in a namespace other than the one where DVO is deployed.  For example, if a service in `some-namespace` wants to scrape the metrics from DVO then a network policy would need to be created like this:

```
oc process --local NAMESPACE='some-namespace' -f deploy/openshift/network-policies.yaml | oc create -f -
```

## Configuring Checks

DVO performs validation checks using kube-linter. The checks configuration is mirrored to the one for the kube-linter project. More information on configuration options can be found [here](https://github.com/stackrox/kube-linter/blob/main/docs/configuring-kubelinter.md), and a list of available checks  can be found [here](https://github.com/stackrox/kube-linter/blob/main/docs/generated/checks.md).

To configure DVO with a different set of checks, create a ConfigMap in the cluster with the new checks configuration. An example of a configuration ConfigMap can be found [here](./deploy/openshift/configmap.yaml).

If no custom configuration is found (the ConfigMap does not exist or does not contain a check declaration), the project sets the checks to the following list by default:
* "host-ipc"
* "host-network"
* "host-pid"
* "non-isolated-pod"
* "pdb-max-unavailable"
* "pdb-min-available"
* "privilege-escalation-container"
* "privileged-container"
* "run-as-non-root"
* "unsafe-sysctls"
* "unset-cpu-requirements"
* "unset-memory-requirements"

**constraint**: Currently, the configuration isn't continuously monitored and is only checked at startup. If a new set of checks is configured in a ConfigMap, the pod running DVO will need to be rebooted.

### Enabling checks

To enable all checks, set the `addAllBuiltIn` property to `true`. If you only want to enable individual checks, include them as a collection in the `include` property and leave `addAllBuiltIn` with a value of `false`.

The `include` property can work together with `doNotAutoAddDefaults` set to `true` in a whitelisting way. Only the checks collection passed in `include` will be executed.

### Disabling checks

To disable all checks, set the `doNotAutoAddDefaults` property to `true`. If you only want to disable individual checks, include them as a collection in the `exclude` property and leave `doNotAutoAddDefaults` with a value of `false`

The `exclude` property takes precedence over the `include` property. If a particular check is in both collections, it will be excluded by default.

The `exclude` property can work in conjunction with `addAllBuiltIn` set to `true` in a blacklisting fashion. All checks will be triggered and only the checks passed in `exclude` will be ignored.

## Tests

You can run the unit tests via

```
make test
```

The end-to-end tests depend on [`ginkgo`](https://onsi.github.io/ginkgo/#installing-ginkgo). After exporting a `KUBECONFIG` variable, it can be ran via

```
make e2e-test
```

We use [openshift boilerplate](https://github.com/openshift/boilerplate) to manage our make targets. See this [doc](https://github.com/openshift/boilerplate/blob/master/boilerplate/openshift/golang-osd-operator/README.md) for further information.


## Releases

To create a new DVO release follow this [New DVO Release](./docs/new-releases.md)

## Roadmap

- e2e tests
