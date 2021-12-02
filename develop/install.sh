#!/usr/bin/env bash

echo "Installing local development toolchain"

if ! command -v k3d &> /dev/null
then
    echo "* installing k3d"
    curl -s https://raw.githubusercontent.com/rancher/k3d/main/install.sh | bash
fi
echo "* k3d installed"

if ! command -v kubectl &> /dev/null
then
    echo "* installing kubectl"
    curl -LO "https://dl.k8s.io/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl.sha256"
    sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
fi
echo "* kubectl installed"

if ! command -v helm &> /dev/null
then
    echo "* installing helm"
    curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
fi
echo "* helm installed"

if ! command -v tilt &> /dev/null
then
    ehco "* installing tilt"
    curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash
fi
echo "* tilt installed"

echo "Adding helm chart repositories"
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts &> /dev/null
helm repo add grafana https://grafana.github.io/helm-charts &> /dev/null

echo "✅ - Install Finished"