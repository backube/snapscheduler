# Project tracking/roadmap

Project tracking is handled via the standard GitHub Issue/PR workflow. To help
with prioritization of issues, open items are tracked on the main [Work items
project board](https://github.com/backube/snapscheduler/projects/1).

The project board provides Kanban-style tracking of ongoing and planned work,
with [project-bot](https://github.com/apps/project-bot) automating the movement
of cards as they progress.

## Longer-term items

Longer-term roadmap items may not yet be tracked on the project board. Such
items include:

### Installation as a cluster-scoped operator

Currently, the scheduler must be installed into each namespace where scheduling
is desired. This is convenient for users (non-admins) who want to quickly add
scheduling to their own namespace(s), but it is not convenient for an
administrator who wishes to make scheduling available cluster-wide.

This would enable installing snapscheduler once for the entire cluster, allowing
it to watch and act on schedules in all namespaces.

### Cluster-wide schedule definitions

This would introduce a the notion of a cluster-scoped snapshot schedule that
could be defined by an admin and made available to all users. For example,
cluster-wide standard schedules like hourly, daily, and weekly could be
pre-defined.

### StorageClass-based schedules

Currently, schedules are applied at the PVC level. However, a StorageClass is
the abstraction where a particular level of storage should be defined. The
"level" of storage should not only be items like performance and reliability but
also data protection, including snapshot policies.

This item would allow allow schedules to be associated with a particular
StorageClass, and all PVC that derive from that class would be snapshotted
according to the associated schedule.

*The loose coupling of PVC to SC as well as the structure of the underlying etcd
database make this a non-straightforward extension.*
