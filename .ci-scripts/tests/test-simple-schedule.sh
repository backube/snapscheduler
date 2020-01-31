#! /bin/bash

set -e -o pipefail

function fail_no_snaps {
    echo "Timeout waiting. No snapshots were created."
    kubectl describe pvc/pvc
    kubectl get snapshotschedule/minute -oyaml
    kubectl -n backube-snapscheduler logs -lapp.kubernetes.io/name=snapscheduler
}

function fail_snap_not_ready {
    echo "Timeout waiting. Snapshot did not become ready"
    echo "Snap name: $1"
    kubectl get "volumesnapshot/$1" -oyaml
    kubectl describe "volumesnapshot/$1"
}

# Create a PVC/PV
cat - <<PVC | kubectl create -f -
---
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: pvc
spec:
  storageClassName: csi-hostpath-sc
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
PVC

# Create a simple schedule
cat - <<SCHED | kubectl create -f -
---
apiVersion: snapscheduler.backube/v1
kind: SnapshotSchedule
metadata:
  name: minute
spec:
  retention:
    maxCount: 5
  schedule: "* * * * *"
  snapshotTemplate:
    snapshotClassName: csi-hostpath-snapclass
SCHED

# Wait for a snapshot to be created from the schedule
DEADLINE=$(( SECONDS + 90 ))
while [[ $(kubectl get volumesnapshots 2> /dev/null | wc -l) -lt 1 ]]; do
    echo "Waiting for snapshots..."
    [[ $SECONDS -lt $DEADLINE ]] || ( fail_no_snaps && exit 1 )
    sleep 10
done

# Get the name of the snapshot
DEADLINE=$(( SECONDS + 120 ))
SNAP_NAME=$(kubectl get volumesnapshots --no-headers | awk '{ print $1 }')
echo "Found snapshot: $SNAP_NAME"

# Wait for it to become "ready"
while [[ $(kubectl get "volumesnapshot/$SNAP_NAME" -ojsonpath="{.status.readyToUse}") != "true" ]]; do
  echo "Waiting for snapshot to be ready..."
    [[ $SECONDS -lt $DEADLINE ]] || ( fail_snap_not_ready "$SNAP_NAME" && exit 1 )
    sleep 10
done
echo "Snapshot $SNAP_NAME is ready!"

# Clean up
kubectl delete SnapshotSchedule/minute
kubectl delete volumesnapshots -l 'snapscheduler.backube/schedule=minute'
kubectl delete pvc/pvc
