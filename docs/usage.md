# Using the scheduler

The scheduler should already be running in the cluster. If not, [go back to
installation](install.md).

## Creating schedules

A snapshot schedule defines:

* A cron-like schedule for taking snapshots
* The set of PVCs that will be selected to snapshot
* The retention policy for the snapshots

### Example schedule

Below is an example snapshot schedule to perform hourly snapshots:

```yaml
---
apiVersion: snapscheduler.backube/v1
kind: SnapshotSchedule
metadata:
  # The name for this schedule. It is also used as a part
  # of the template for naming the snapshots.
  name: hourly
  # Schedules are namespaced objects
  namespace: myns
spec:
  # A LabelSelector to control which PVCs should be snapshotted
  claimSelector:  # optional
  # Set to true to make the schedule inactive
  disabled: false  # optional
  retention:
    # The length of time a given snapshot should be
    # retained, specified in hours. (168h = 1 week)
    expires: "168h"  # optional
    # The maximum number of snapshots per PVC to keep
    maxCount: 10  # optional
  # The cronspec (https://en.wikipedia.org/wiki/Cron#Overview)
  # that defines the schedule. The following pre-defined
  # shortcuts are also supported: @hourly, @daily, @weekly,
  # @monthly, and @yearly
  schedule: "0 * * * *"
  snapshotTemplate:
    # A set of labels can be added to each
    # VolumeSnapshot object
    labels:  # optional
      mylabel: myvalue
    # The SnapshotClassName to use when creating the
    # snapshots. If omitted, the cluster default will
    # be used.
    snapshotClassName: ebs-csi  # optional
```

### Selecting PVCs

The `spec.claimSelector` is an optional field can be used to limit which PVCs
are snapshotted according to the schedule. This field is a
`metav1.LabelSelector`. Please see the kubernetes documentation on [label
selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors)
for a full explanation.

The claim selector supports both the simple `matchLabels` as well as the newer
set-based matching. For example, to use matchLabels, include the following:

```yaml
spec:
  claimSelector:
    matchLabels:
      thislabel: that
```

Including the above in the schedule would limit the schedule to only PVCs that
carry a label of `thislabel: that` in their `metadata.labels` list.

## Viewing schedules

The existing schedules can be viewed by:

```console
$ kubectl -n myns get snapshotschedules
NAME     SCHEDULE    MAX AGE   MAX NUM   DISABLED   NEXT SNAPSHOT
hourly   0 * * * *   168h      10                   2019-11-01T20:00:00Z
```

## Snapshots

The snapshots that are created by a schedule are named by the following
template: `<pvc_name>-<schedule_name>-<snapshot_time>`

The example, below, shows two snapshots of the PVC named `data` which were taken
by the `hourly` schedule. The time of these two snapshots is visible in the
`YYYYMMDDHHMM` format, UTC timezone.

```console
$ kubectl -n myns get volumesnapshots
NAME                       AGE
data-hourly-201911011900   82m
data-hourly-201911012000   22m
```
