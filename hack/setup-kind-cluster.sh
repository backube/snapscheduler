#! /bin/bash

set -e -o pipefail

# Possible versions:
# https://hub.docker.com/r/kindest/node/tags?page=1&ordering=name
# skopeo inspect docker://kindest/node:v1.17.0 | jq .RepoTags
# 1.13.12, 1.14.10, 1.15.7, 1.16.4, 1.17.0
KUBE_VERSION="${1:-1.17.0}"

# Determine the Kube minor version
[[ "${KUBE_VERSION}" =~ ^[0-9]+\.([0-9]+) ]] && KUBE_MINOR="${BASH_REMATCH[1]}" || exit 1

KIND_CONFIG=""
KIND_CONFIG_FILE="$(mktemp --tmpdir kind-config-XXXXXX.yaml)"

if [[ $KUBE_MINOR -lt 17 ]]; then
  KIND_CONFIG="--config ${KIND_CONFIG_FILE}"
  cat - > "${KIND_CONFIG_FILE}" <<KINDCONFIG
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
kubeadmConfigPatches:
- |
  kind: ClusterConfiguration
  metadata:
    name: config
  apiServer:
    extraArgs:
      "feature-gates": "VolumeSnapshotDataSource=true"
  scheduler:
    extraArgs:
      "feature-gates": "VolumeSnapshotDataSource=true"
  controllerManager:
    extraArgs:
      "feature-gates": "VolumeSnapshotDataSource=true"
- |
  kind: InitConfiguration
  metadata:
    name: config
  nodeRegistration:
    kubeletExtraArgs:
      "feature-gates": "VolumeSnapshotDataSource=true"
- |
  kind: KubeletConfiguration
  featureGates:
    VolumeSnapshotDataSource: true
KINDCONFIG
fi

# Create the cluster
kind delete cluster || true
# shellcheck disable=SC2086
kind create cluster ${KIND_CONFIG} --image "kindest/node:v${KUBE_VERSION}"

rm -f "${KIND_CONFIG_FILE}"

# Kube >= 1.17, we need to deploy the snapshot controller
if [[ $KUBE_MINOR -ge 17 ]]; then
        TAG="v2.0.1"
        kubectl create -f "https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${TAG}/config/crd/snapshot.storage.k8s.io_volumesnapshotclasses.yaml"
        kubectl create -f "https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${TAG}/config/crd/snapshot.storage.k8s.io_volumesnapshotcontents.yaml"
        kubectl create -f "https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${TAG}/config/crd/snapshot.storage.k8s.io_volumesnapshots.yaml"

        kubectl create -f "https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${TAG}/deploy/kubernetes/snapshot-controller/rbac-snapshot-controller.yaml"
        kubectl create -f "https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${TAG}/deploy/kubernetes/snapshot-controller/setup-snapshot-controller.yaml"
fi

# Install the hostpath CSI driver
HP_BASE="$(mktemp --tmpdir -d csi-driver-host-path-XXXXXX)"
git clone --depth 1 https://github.com/kubernetes-csi/csi-driver-host-path.git "$HP_BASE"
if [[ $KUBE_MINOR -eq 14 ]]; then
        cd "$HP_BASE"
        git pull --unshallow && git pull
        git checkout "v1.2.0"
fi
if [[ $KUBE_MINOR -lt 14 ]]; then
        cd "$HP_BASE"
        git pull --unshallow && git pull
        git checkout "v1.1.0"
fi
"${HP_BASE}/deploy/kubernetes-1.$KUBE_MINOR"/deploy-hostpath.sh
rm -rf "${HP_BASE}"

kubectl apply -f - <<SC
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: csi-hostpath-sc
provisioner: hostpath.csi.k8s.io
reclaimPolicy: Delete
volumeBindingMode: Immediate
allowVolumeExpansion: true
SC

# Change the default SC
kubectl annotate sc/standard storageclass.kubernetes.io/is-default-class-
kubectl annotate sc/csi-hostpath-sc storageclass.kubernetes.io/is-default-class="true"

# For some versions we need to create the snapclass ourselves
if [[ $KUBE_MINOR -eq 15 || $KUBE_MINOR -eq 16 ]]; then
        kubectl create -f - <<SNAPALPHA
apiVersion: snapshot.storage.k8s.io/v1alpha1
kind: VolumeSnapshotClass
metadata:
  name: csi-hostpath-snapclass
snapshotter: hostpath.csi.k8s.io
SNAPALPHA
fi
