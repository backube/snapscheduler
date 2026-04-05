#! /bin/sh

set -e

KIND_VERSION="0.31.0"
KUBERNETES_VERSION="1.33.0"

# Create a temporary directory that matches the one from the devcontainer.json file
mkdir -p "$TMPDIR"

# Configure docker to use the "legacy" image store until `kind load image` works w/ containerd
# See https://github.com/kubernetes-sigs/kind/issues/3795
cat - <<EOF | sudo tee /etc/docker/daemon.json
{
  "features": {
    "containerd-snapshotter": false
    }
}
EOF
sudo service docker stop  # restart doesn't work
sudo killall dockerd  # stop the docker daemon
sudo /usr/local/share/docker-init.sh  # Restart docker to apply the new configuration

# Install kubectl
curl -LO "https://dl.k8s.io/release/v${KUBERNETES_VERSION}/bin/linux/amd64/kubectl"
sudo install ./kubectl /usr/local/bin/ && rm kubectl
kubectl version --client

# Install kind
curl -L -o kind https://github.com/kubernetes-sigs/kind/releases/download/v${KIND_VERSION}/kind-linux-amd64
sudo install ./kind /usr/local/bin && rm kind
kind version

make kuttl
sudo install ./bin/kuttl /usr/local/bin/
make helm
sudo install ./bin/helm /usr/local/bin/
