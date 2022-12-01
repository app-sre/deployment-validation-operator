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
    - Copy and Paste the pre-existing DVO version directory (ex. 0.2.0, 0.2.1) and change the name of the directory to reflect the new release version
    - Modify the clusterserviceversion file's name within the directory to reflect the new release version
    
    ```yaml
    # Edit the clusterserviceversion file within the directory and modify the following lines to reflect the new release
    # RELEASE VERSION == 0.2.0, 0.2.1, etc.

    * metadata.annotations.containerImage: quay.io/deployment-validation-operator/dv-operator:<RELEASE VERSION>
    * metadata.name: deployment-validation-operator.v<RELEASE VERSION>
    * spec.install.spec.deployments.spec.template.spec.containers.image: quay.io/deployment-validation-operator/dv-operator:<RELEASE VERSION>
    * spec.links.url: https://quay.io/deployment-validation-operator/dv-operator:<RELEASE VERSION>
    * spec.version: <RELEASE VERSION>
    * spec.skipRange: '>=0.0.10 <X.Y.Z' X.Y.Z being the new RELEASE VERSION

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