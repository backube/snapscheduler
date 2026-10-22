#! /bin/bash

set -e -o pipefail

# Possible versions:
# https://hub.docker.com/r/kindest/node/tags?page=1&ordering=name
# skopeo list-tags docker://kindest/node
KUBE_VERSION="${1:-1.33.0}"

function log {
  echo "=====  $*  ====="
}

# Determine the Kube minor version
[[ "${KUBE_VERSION}" =~ ^[0-9]+\.([0-9]+) ]] && KUBE_MINOR="${BASH_REMATCH[1]}" || exit 1
log "Detected kubernetes minor version: ${KUBE_MINOR}"

KIND_CONFIG=""
KIND_CONFIG_FILE="$(mktemp --tmpdir kind-config-XXXXXX.yaml)"

# Enforce pod security standards
# https://kubernetes.io/docs/tasks/configure-pod-container/enforce-standards-admission-controller/#configure-the-admission-controller
# https://kubernetes.io/docs/tutorials/security/cluster-level-pss/
# PodSecurity admission is available in Kubernetes 1.23 and newer.
if [[ $KUBE_MINOR -ge 23 ]]; then
  log "Setting up PodSecurity admission"
  # PodSecurityConfiguration v1 is available in 1.25+; 1.23-1.24 use v1beta1.
  if [[ $KUBE_MINOR -ge 25 ]]; then
    PSA_API_VERSION="v1"
  else
    PSA_API_VERSION="v1beta1"
  fi
  # Pod security controller has to be configured by passing a config file to
  # api-server on the command line
  SHARED_DIR="$(mktemp -d --tmpdir control-plane-shared-XXXXXX)"
  cat - > "${SHARED_DIR}/admission.yaml" <<ADMCFG
apiVersion: apiserver.config.k8s.io/v1
kind: AdmissionConfiguration
plugins:
- name: PodSecurity
  configuration:
    apiVersion: pod-security.admission.config.k8s.io/${PSA_API_VERSION}
    kind: PodSecurityConfiguration
    # Defaults applied when a mode label is not set.
    #
    # Level label values must be one of:
    # - "privileged" (default)
    # - "baseline"
    # - "restricted"
    #
    # Version label values must be one of:
    # - "latest" (default)
    # - specific version like "v1.24"
    defaults:
      enforce: "restricted"
      enforce-version: "latest"
      audit: "restricted"
      audit-version: "latest"
      warn: "restricted"
      warn-version: "latest"
    exemptions:
      usernames: []
      runtimeClasses: []
      namespaces:
        # KUTTL tests use per-test namespaces and remain enforced.
        - default             # CSI hostpath runs here
        - kube-system
        - local-path-storage  # default local storage provisioner
ADMCFG
  KIND_CONFIG="--config ${KIND_CONFIG_FILE}"
  cat - > "${KIND_CONFIG_FILE}" <<KINDCONFIG
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: ClusterConfiguration
    apiServer:
        extraArgs:
          admission-control-config-file: /etc/config/admission.yaml
        extraVolumes:
          - name: config-files
            hostPath: /shared
            mountPath: /etc/config
            readOnly: false
            pathType: "DirectoryOrCreate"
  extraMounts:
  - hostPath: ${SHARED_DIR}
    containerPath: /shared
    readOnly: true
    selinuxRelabel: false
KINDCONFIG
else
  log "PodSecurity admission is unavailable before Kubernetes 1.23; skipping PSA setup and enforcement probe"
fi

if [[ $KUBE_MINOR -le 16 ]]; then
  KIND_CONFIG="--config ${KIND_CONFIG_FILE}"
  cat - > "${KIND_CONFIG_FILE}" <<KINDCONFIG16
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
KINDCONFIG16
fi

if [[ $KUBE_MINOR -le 13 ]]; then
  KIND_CONFIG="--config ${KIND_CONFIG_FILE}"
  cat - > "${KIND_CONFIG_FILE}" <<KINDCONFIG13
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
kubeadmConfigPatches:
- |
  kind: ClusterConfiguration
  metadata:
    name: config
  apiServer:
    extraArgs:
      "feature-gates": "CSIDriverRegistry=true,CSINodeInfo=true,VolumeSnapshotDataSource=true"
  scheduler:
    extraArgs:
      "feature-gates": "CSIDriverRegistry=true,CSINodeInfo=true,VolumeSnapshotDataSource=true"
  controllerManager:
    extraArgs:
      "feature-gates": "CSIDriverRegistry=true,CSINodeInfo=true,VolumeSnapshotDataSource=true"
- |
  kind: InitConfiguration
  metadata:
    name: config
  nodeRegistration:
    kubeletExtraArgs:
      "feature-gates": "CSIDriverRegistry=true,CSINodeInfo=true,VolumeSnapshotDataSource=true"
- |
  kind: KubeletConfiguration
  featureGates:
    CSIDriverRegistry: true
    CSINodeInfo: true
    VolumeSnapshotDataSource: true
KINDCONFIG13
fi

# Create the cluster
kind delete cluster || true
# shellcheck disable=SC2086
kind create cluster ${KIND_CONFIG} --image "kindest/node:v${KUBE_VERSION}"

rm -f "${KIND_CONFIG_FILE}"

if [[ $KUBE_MINOR -ge 23 ]]; then
  # Use a dedicated namespace that is not in the PodSecurity exemption list.
  PSA_PROBE_NAMESPACE="psa-enforcement-probe-$$"
  kubectl create namespace "${PSA_PROBE_NAMESPACE}" >/dev/null

  cleanup_psa_probe_namespace() {
    kubectl delete namespace "${PSA_PROBE_NAMESPACE}" --ignore-not-found --wait=false >/dev/null 2>&1 || true
  }
  trap cleanup_psa_probe_namespace EXIT

  log "Waiting for default ServiceAccount in ${PSA_PROBE_NAMESPACE}"
  PSA_PROBE_SERVICE_ACCOUNT=""
  for ((PSA_PROBE_SA_WAITED = 0; PSA_PROBE_SA_WAITED <= 30; PSA_PROBE_SA_WAITED++)); do
    PSA_PROBE_SERVICE_ACCOUNT="$(kubectl get serviceaccount/default -n "${PSA_PROBE_NAMESPACE}" --ignore-not-found -o name)"
    if [[ -n "${PSA_PROBE_SERVICE_ACCOUNT}" ]]; then
      break
    fi
    if (( PSA_PROBE_SA_WAITED < 30 )); then
      sleep 1
    fi
  done
  if [[ -z "${PSA_PROBE_SERVICE_ACCOUNT}" ]]; then
    log "ERROR: timed out waiting 30 seconds for serviceaccount/default in ${PSA_PROBE_NAMESPACE}"
    exit 1
  fi

  log "Verifying PodSecurity restricted enforcement"
  if PSA_PROBE_OUTPUT="$(kubectl create --dry-run=server -f - 2>&1 <<POD
apiVersion: v1
kind: Pod
metadata:
  name: psa-enforcement-probe
  namespace: ${PSA_PROBE_NAMESPACE}
spec:
  containers:
  - name: probe
    image: registry.k8s.io/pause:3.10
    securityContext:
      privileged: true
POD
)"; then
    log "ERROR: PodSecurity accepted a privileged Pod in the probe namespace"
    echo "${PSA_PROBE_OUTPUT}" >&2
    exit 1
  elif ! grep -Fq 'violates PodSecurity "restricted' <<<"${PSA_PROBE_OUTPUT}"; then
    log "ERROR: PodSecurity enforcement probe failed for an unexpected reason"
    echo "${PSA_PROBE_OUTPUT}" >&2
    exit 1
  fi

  log "PodSecurity restricted enforcement verified"
  kubectl delete namespace "${PSA_PROBE_NAMESPACE}" --wait=true >/dev/null
  trap - EXIT
fi

# Kube >= 1.17, we need to deploy the snapshot controller
if [[ $KUBE_MINOR -ge 25 ]]; then  # Starting w/ v8, min supported kube is 1.25
  # renovate: datasource=github-releases depName=kubernetes-csi/external-snapshotter versioning=semver-coerced
  TAG="v8.7.0"  # https://github.com/kubernetes-csi/external-snapshotter/releases
  log "Deploying external snapshotter: ${TAG}"
  kubectl create -k "https://github.com/kubernetes-csi/external-snapshotter/client/config/crd?ref=${TAG}"
  kubectl create -n kube-system -k "https://github.com/kubernetes-csi/external-snapshotter/deploy/kubernetes/snapshot-controller?ref=${TAG}"
elif [[ $KUBE_MINOR -ge 24 ]]; then  # Kube 1.24 removed snapshot.storage.k8s.io/v1beta1
  TAG="v7.0.2"  # https://github.com/kubernetes-csi/external-snapshotter/releases
  log "Deploying external snapshotter: ${TAG}"
  kubectl create -k "https://github.com/kubernetes-csi/external-snapshotter/client/config/crd?ref=${TAG}"
  kubectl create -n kube-system -k "https://github.com/kubernetes-csi/external-snapshotter/deploy/kubernetes/snapshot-controller?ref=${TAG}"
elif [[ $KUBE_MINOR -ge 20 ]]; then  # Kube 1.20 added snapshot.storage.k8s.io/v1
  TAG="v5.0.1"  # https://github.com/kubernetes-csi/external-snapshotter/releases
  log "Deploying external snapshotter: ${TAG}"
  kubectl create -k "https://github.com/kubernetes-csi/external-snapshotter/client/config/crd?ref=${TAG}"
  kubectl create -n kube-system -k "https://github.com/kubernetes-csi/external-snapshotter/deploy/kubernetes/snapshot-controller?ref=${TAG}"
elif [[ $KUBE_MINOR -ge 17 ]]; then  # Kube 1.17 switched snapshot.storage.k8s.io/v1alpha1 -> v1beta1
  TAG="v3.0.3"  # https://github.com/kubernetes-csi/external-snapshotter/releases
  log "Deploying external snapshotter: ${TAG}"
  kubectl create -f "https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${TAG}/client/config/crd/snapshot.storage.k8s.io_volumesnapshotclasses.yaml"
  kubectl create -f "https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${TAG}/client/config/crd/snapshot.storage.k8s.io_volumesnapshotcontents.yaml"
  kubectl create -f "https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${TAG}/client/config/crd/snapshot.storage.k8s.io_volumesnapshots.yaml"

  kubectl create -f "https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${TAG}/deploy/kubernetes/snapshot-controller/rbac-snapshot-controller.yaml"
  kubectl create -f "https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${TAG}/deploy/kubernetes/snapshot-controller/setup-snapshot-controller.yaml"
fi

# Kube 1.13 requires CSIDriver & CSINodeInfo CRDs
if [[ $KUBE_MINOR -eq 13 ]]; then
  kubectl create -f https://raw.githubusercontent.com/kubernetes/csi-api/master/pkg/crd/manifests/csidriver.yaml
  kubectl create -f https://raw.githubusercontent.com/kubernetes/csi-api/master/pkg/crd/manifests/csinodeinfo.yaml
fi

# Install the hostpath CSI driver
# https://github.com/kubernetes-csi/csi-driver-host-path/releases
HP_BASE="$(mktemp --tmpdir -d csi-driver-host-path-XXXXXX)"
case "$KUBE_MINOR" in
  13)
    TAG="v1.1.0"
    DEPLOY_SCRIPT="deploy-hostpath.sh"
    ;;
  14)
    TAG="v1.2.0"
    DEPLOY_SCRIPT="deploy-hostpath.sh"
    ;;
  15)
    TAG="v1.3.0"
    DEPLOY_SCRIPT="deploy-hostpath.sh"
    ;;
  16|17)
    TAG="v1.4.0"
    DEPLOY_SCRIPT="deploy.sh"
    ;;
  18)
    TAG="v1.7.2"
    DEPLOY_SCRIPT="deploy.sh"
    ;;
  19|20)
    TAG="v1.7.3"
    DEPLOY_SCRIPT="deploy.sh"
    ;;
  21)
    TAG="v1.9.0"
    DEPLOY_SCRIPT="deploy.sh"
    ;;
  22|23)
    TAG="v1.10.0"
    DEPLOY_SCRIPT="deploy.sh"
    ;;
  24)
    TAG="v1.12.1"
    DEPLOY_SCRIPT="deploy.sh"
    ;;
  *)
    # renovate: datasource=github-releases depName=kubernetes-csi/csi-driver-host-path versioning=semver-coerced
    TAG="v1.19.0"
    DEPLOY_SCRIPT="deploy.sh"
    ;;
esac
log "Deploying CSI hostpath driver: ${TAG}"
git clone --depth 1 -b "$TAG" https://github.com/kubernetes-csi/csi-driver-host-path.git "$HP_BASE"

DEPLOY_PATH="${HP_BASE}/deploy/kubernetes-1.${KUBE_MINOR}/"
# For versions not yet supported, use the latest
if [[ ! -d "${DEPLOY_PATH}" ]]; then
  DEPLOY_PATH="${HP_BASE}/deploy/kubernetes-latest/"
fi

# Remove the CSI testing manifest. It exposes csi.sock as a TCP socket using
# socat. This is insecure, but the larger problem is that it pulls the socat
# image from docker.io, making it subject to rate limits that break this script.
rm -f "${DEPLOY_PATH}/hostpath/csi-hostpath-testing.yaml"

"${DEPLOY_PATH}/${DEPLOY_SCRIPT}"
rm -rf "${HP_BASE}"

CSI_DRIVER_NAME="hostpath.csi.k8s.io"
if [[ $KUBE_MINOR -eq 13 ]]; then
  CSI_DRIVER_NAME="csi-hostpath"
fi
log "Creating StorageClass for CSI hostpath driver"
kubectl apply -f - <<SC
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: csi-hostpath-sc
provisioner: ${CSI_DRIVER_NAME}
reclaimPolicy: Delete
volumeBindingMode: Immediate
allowVolumeExpansion: true
SC

# Change the default SC
kubectl annotate sc/standard storageclass.kubernetes.io/is-default-class-
kubectl annotate sc/csi-hostpath-sc storageclass.kubernetes.io/is-default-class="true"

# For some versions we need to create the snapclass ourselves
if [[ $KUBE_MINOR -eq 15 || $KUBE_MINOR -eq 16 ]]; then
  log "Creating VolumeStorageClass for CSI hostpath driver"
  kubectl create -f - <<SNAPALPHA
apiVersion: snapshot.storage.k8s.io/v1alpha1
kind: VolumeSnapshotClass
metadata:
  name: csi-hostpath-snapclass
snapshotter: hostpath.csi.k8s.io
SNAPALPHA
fi

# Make VSC the cluster default
kubectl annotate volumesnapshotclass/csi-hostpath-snapclass snapshot.storage.kubernetes.io/is-default-class="true"

# On 1.20, we need to enable fsGroup so that volumes are writable
# Release v1.9.0 included this change for 1.21+
if [[ $KUBE_MINOR -eq 20 ]]; then
  log "Enabling fsGroupPolicy for CSI driver"
  kubectl delete csidriver hostpath.csi.k8s.io
  kubectl create -f - <<DRIVERINFO
apiVersion: storage.k8s.io/v1
kind: CSIDriver
metadata:
  name: hostpath.csi.k8s.io
  labels:
    app.kubernetes.io/instance: hostpath.csi.k8s.io
    app.kubernetes.io/part-of: csi-driver-host-path
    app.kubernetes.io/name: hostpath.csi.k8s.io
    app.kubernetes.io/component: csi-driver
spec:
  volumeLifecycleModes:
  - Persistent
  - Ephemeral
  podInfoOnMount: true
  fsGroupPolicy: File
DRIVERINFO
fi

# Add a node topology key so that e2e tests can run
kubectl label nodes --all topology.kubernetes.io/zone=z1
