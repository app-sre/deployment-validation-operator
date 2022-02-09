# Deployment Validation Operator

## Description

The Deployment Validation Operator (DVO) checks deployments and other resources against a curated collection of best practices.

These best practices focus mainly on ensuring that the applications are fault-tolerant.

DVO will only monitor Kubernetes resources and will not modify them in any way. Instead, it will report failed validations via Prometheus, which will allow users of this operator to create alerts based on its results. All the metrics are gauges that will report `1` if the best-practice has failed. The metric will always have three parameters: `name`, `namespace` and `kind`.

This operator doesn't define any CRDs at the moment. It has been bootstrapped with `operator-sdk` making it possible to add a CRD in the future if required.

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

oc process --local NAMESPACE_IGNORE_PATTERN='openshift.*|kube-.+' -f deploy/openshift/deployment-validation-operator-olm.yaml | oc create -f -
```

```
# otherwise, deploy this if you DO want OLM to automatically upgrade DVO
# set REGISTRY_POLLING_INTERVAL to be shorter to have OLM check for new DVO versions more frequently if desired; e.g. '45m'
# the shorter the interval, the more resources OLM may consume
# read more about OLM catalog polling: https://github.com/operator-framework/
operator-lifecycle-manager/blob/master/doc/design/catalog-polling.md

oc process --local \
NAMESPACE_IGNORE_PATTERN='openshift.*|kube-.+' \
REGISTRY_POLLING_INTERVAL='24h' \
-f deploy/openshift/deployment-validation-operator-olm-with-polling.yaml \
| oc create -f -
```

If DVO is deployed to a namespace other than the one where OLM is deployed, which is usually the case, then a network policy may be required to allow OLM to see the artifacts in the DVO namespace.  For example, if OLM is deployed in the namespace `operator-lifecycle-manager` then the network policy would be deployed like this:

```
oc process --local NAMESPACE='operator-lifecycle-manager' -f deploy/openshift/network-policies.yaml | oc create -f -
```

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

## Tests

You can run the unit tests via

```
make test
```

We use [openshift boilerplate](https://github.com/openshift/boilerplate) to manage our make targets. See this [doc](https://github.com/openshift/boilerplate/blob/master/boilerplate/openshift/golang-osd-operator/README.md) for further information.


## Releases

### New Release

**Before Proceeding:** 
* Assure desired changes for new release have been submitted and merged successfully
* Check with team to verify if this is a MAJOR, MINOR, or a PATCH release

**Release Process:**
1. Create a new DVO release in GitHub 
    
    - Create new release on GitHub page from the right column
    - Follow the model of Major/Minor/Patch (x/y/z, 0.1.2) 
    - Provide a description of the release (Auto-generate or Manually)

2. Publish the new DVO release to Quay.io (no action required) 
    
    - Generating a new tagged release in GitHub will trigger a jenkins job that will build a new image of DVO with the new release tag
    - Verify Jenkins Job was successful - [DVO Jenkins](https://ci.int.devshift.net/view/deployment-validation-operator/job/app-sre-deployment-validation-operator-gh-build-tag/)

3. Publish new DVO release to Operator-Hub

    - OperatorHub Repository for DVO - [DVO OLM](https://github.com/k8s-operatorhub/community-operators/tree/main/operators/deployment-validation-operator)
    - Edit deployment-validation-operator.package.yaml to reflect the new release
    
    ```yaml
    # RELEASE VERSION == 0.2.0, 0.2.1, etc.
    
    * channels.currentCSV: deployment-validation-operator.v<RELEASE VERSION>
    ```
    
    - As a shortcut, you may choose to copy+paste the most recent already-existing DVO version directory (ex. 0.2.0, 0.2.1) and change the name of the directory to reflect the new release version
    - Modify the clusterserviceversion file's name within the directory to reflect the new release version
    
    ```yaml
    # Edit the clusterserviceversion file within the directory and modify the following lines to reflect the new release
    # RELEASE VERSION == 0.2.0, 0.2.1, etc.

    * metadata.annotations.containerImage: quay.io/deployment-validation-operator/dv-operator:<RELEASE VERSION>
    * metadata.name: deployment-validation-operator.v<RELEASE VERSION>
    * spec.install.spec.deployments.spec.template.spec.containers.image: quay.io/deployment-validation-operator/dv-operator:<RELEASE VERSION>
    * spec.links.url: https://quay.io/deployment-validation-operator/dv-operator:<RELEASE VERSION>
    * spec.version: <RELEASE VERSION>

    # Modify the following line to reflect the previous release version for upgrade purposes 
    # (ex. If going from 0.2.1 -> 0.2.2, then the previous release was 0.2.1)

    * spec.replaces: deployment-validation-operator.v<PREVIOUS RELEASE VERSION>
    ```

    - If changes need to be made to add/subtract reviewers, this can be changed within `ci.yaml`
        * This file allows for authorized users to review the PRs pushed to the DVO OLM project

    - If need-be for the nature of what the changes in the new DVO release, update the rest of these files accordingly

    - Submit a PR

4. OLM updates DVO version across DVO-consuming kubernetes clusters (no action required)

    - (Right now DVO is in an alpha-state, and so clusters running an OLM that is configured to ignore alpha releases in Operator-Hub may have unreliable success with the following):

    - Once the merge request to the `k8s-operatorhub/community-operators` GitHub repo is merged, the latest version of DVO available through the Operator-Hub ecosystem should automatically update. You can check the latest version available [here](https://operatorhub.io/operator/deployment-validation-operator).


## Roadmap

- e2e tests
