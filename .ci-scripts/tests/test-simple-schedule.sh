#! /bin/bash

set -e -o pipefail

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
    maxCount: 2
  schedule: "* * * * *"
  snapshotTemplate:
    snapshotClassName: csi-hostpath-snapclass
SCHED

# Wait for a snapshot to be created from the schedule
DEADLINE=$(( SECONDS + 90 ))
while [[ $(kubectl get volumesnapshots 2> /dev/null | wc -l) -lt 1 ]]; do
    echo "Waiting for snapshots..."
    [[ $SECONDS -lt $DEADLINE ]] || ( echo "Timeout waiting. No snapshots were created." && exit 1 )
    sleep 10
done

kubectl delete SnapshotSchedule/minute
kubectl delete volumesnapshots -l 'snapscheduler.backube/schedule=minute'
kubectl delete pvc/pvc
