#!/usr/bin/env bash

CLUSTER_NAME="dvo-local"
PROMETHEUS_NAMESPACE="prom"
GRAFANA_NAMESPACE="graf"
DEVELOPMENT_NAMESPACE="check"
DVO_NAMESPACE="deployment-validation-operator"

INSTALL_PROMETHEUS="true"
INSTALL_PROMETHEUS_OPERATOR="false"
INSTALL_GRAFANA="false"

k3d cluster list | grep ${CLUSTER_NAME} &> /dev/null
SUCCESS=$?
if [ $SUCCESS -eq 1 ]
then
    echo "Creating ${CLUSTER_NAME} k3d cluster"
    k3d cluster create $CLUSTER_NAME -p "8080:30080@server:0" -p "8081:30081@server:0" --registry-create $CLUSTER_NAME-registry:0.0.0.0:5432 --registry-config registries.yaml
    SUCCESS=$?
    if [ $SUCCESS -eq 1 ]
    then
        echo "Error creating k3d cluster"
        exit 1
    fi
    echo ""
fi

CURRENT_CONTEXT=$(kubectl config current-context)
if [ "$CURRENT_CONTEXT" != "k3d-$CLUSTER_NAME" ]
then
    kubectl config use-context $CLUSTER_NAME
    SUCCESS=$?
    if [ $SUCCESS -eq 1 ]
    then
        echo "Error selecting k3d kubernetes context"
        exit 1
    fi
fi

if [ "$INSTALL_PROMETHEUS" == "true" ]
then
    echo "Installing prometheus"
    echo ""
    helm install -n $PROMETHEUS_NAMESPACE --create-namespace --set server.service.type=NodePort --set server.service.nodePort=30080 prom prometheus-community/prometheus &> /dev/null
fi

if [ "$INSTALL_PROMETHEUS_OPERATOR" == "true" ]
then
    echo "Installing prometheus-operator"
    echo ""
    helm install -n $PROMETHEUS_NAMESPACE prom-operator prometheus-community/prometheus-operator &> /dev/null
fi

if [ "$INSTALL_GRAFANA" == "true" ]
then
    echo "Installing grafana"
    echo ""
    helm install -n $GRAFANA_NAMESPACE --create-namespace --set service.type=NodePort --set service.nodePort=30081 --set   graf grafana/grafana &> /dev/null
fi

echo "Installing deployment validation operator"
kubectl create namespace $DVO_NAMESPACE
for file in ../deploy/kubernetes/*
do
    if [ "$file" != "../deploy/kubernetes/operator.yaml" ]
    then
        kubectl apply -f $file
    fi 
done
kubectl apply -f manifests/cluster-role-binding.yaml
kubectl apply -f manifests/role-binding.yaml
echo ""

echo "Creating nginx deployments in $DEVELOPMENT_NAMESPACE namespace"
kubectl create namespace $DEVELOPMENT_NAMESPACE
kubectl create deployment --image nginx nginx -n $DEVELOPMENT_NAMESPACE
echo ""

echo "✅ - Bootstrap Finished"
echo "Run $ tilt up to build and deploy DVO from your local repository."
echo "Saving new changes to any DVO code file will trigger an automated rebuild and redeploy."