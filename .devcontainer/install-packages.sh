#! /bin/sh

set -e

KIND_VERSION="0.31.0"
KUBERNETES_VERSION="1.33.0"

# Create a temporary directory that matches the one from the devcontainer.json file
sudo mkdir -p "$TMPDIR"
sudo chmod 1777 "$TMPDIR"

# Configure docker to use the "legacy" image store until `kind load image` works w/ containerd
# See https://github.com/kubernetes-sigs/kind/issues/3795
#
# Also, switch to nftables to avoid iptables errors in Docker.
# This fixes "Table does not exist (do you need to insmod?)" errors when starting dockerd.
sudo update-alternatives --set iptables /usr/sbin/iptables-nft

cat - <<EOF | sudo tee /etc/docker/daemon.json
{
  "features": {
    "containerd-snapshotter": false
    }
}
EOF

# Restart dockerd to apply the new configuration.
# We stop it forcefully to ensure a clean slate, then use the feature's script to restart it.
sudo service docker stop || true
sudo pkill -x dockerd || true
sudo pkill -x containerd || true
sudo /usr/local/share/docker-init.sh

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
