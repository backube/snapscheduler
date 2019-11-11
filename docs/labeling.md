# Labeling strategies

PVCs are selected for snapshotting via labels. Each `snapshotschedule` has a
label selector that is used to determine which PVCs are subject to the schedule.
There are a number of different strategies that can be employed, several of
which are described below. These are just suggestions to consider and can be
customized as necessary to fit within a given environment.

## Application-centric labeling

This labeling approach is best suited to situations where an application's data
is tagged with a common set of labels (e.g., `app=myapp`), and a schedule is
being defined specifically for that application.

In this case, the application's label can be directly incorporated into a custom
schedule for that application:

```yaml
---
apiVersion: snapscheduler.backube/v1alpha1
kind: SnapshotSchedule
metadata:
  name: myapp
spec:
  claimSelector:
    matchLabels:
      app: myapp
  # ...other fields omitted...
```

The main benefit to this method is that the application's manifests do not need
to be modified.

## Schedule-centric labeling

This labeling approach best for situations where it is desirable to have a
standard set of schedules that can be used by different applications in an ad
hoc manner.

Schedules can be defined with their own unique label:

```yaml
---
apiVersion: snapscheduler.backube/v1alpha1
kind: SnapshotSchedule
metadata:
  name: hourly
spec:
  claimSelector:
    matchLabels:
      "schedule/hourly": "enabled"
  schedule: "@hourly"

---
apiVersion: snapscheduler.backube/v1alpha1
kind: SnapshotSchedule
metadata:
  name: daily
spec:
  claimSelector:
    matchLabels:
      "schedule/daily": "enabled"
  schedule: "@daily"
```

Individual PVCs can be tagged to use one or more of these standard schedules by
including the appropriate label(s):

```yaml
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: mydata
  labels:
    "schedule/hourly": "enabled"
    "schedule/daily": "enabled"
spec:
  # ...omitted...
```

## Service-level labeling

Building on the above example, a class of service for snapshot-based protection
could be defined. For example, it is possible to define a "gold" tier that
provides:

- 6 hourly
- 7 daily
- 4 weekly

```yaml
---
apiVersion: snapscheduler.backube/v1alpha1
kind: SnapshotSchedule
metadata:
  name: gold-hourly
spec:
  claimSelector:
    matchLabels:
      "snapshot-tier" "gold"
  retention:
    maxCount: 6
  schedule: "@hourly"

---
apiVersion: snapscheduler.backube/v1alpha1
kind: SnapshotSchedule
metadata:
  name: gold-daily
spec:
  claimSelector:
    matchLabels:
      "snapshot-tier" "gold"
  retention:
    maxCount: 7
  schedule: "@daily"

---
apiVersion: snapscheduler.backube/v1alpha1
kind: SnapshotSchedule
metadata:
  name: gold-weekly
spec:
  claimSelector:
    matchLabels:
      "snapshot-tier" "gold"
  retention:
    maxCount: 4
  schedule: "@weekly"
```

A PVC can then reference the snapshot tier:

```yaml
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: mydata
  labels:
    "snapshot-tier": "gold"
spec:
  # ...omitted...
```
