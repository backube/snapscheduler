# Updating the operator SDK version

Check out the `sdk-scaffold` branch

```
$ git checkout sdk-scaffold
Switched to branch 'sdk-scaffold'
```

Move the existing tree out of the way

```
$ cd ..
$ mv SnapScheduler SnapScheduler_tmp
$
```

Generate the scaffolding files

```
$ operator-sdk new SnapScheduler --cluster-scoped --skip-git-init
INFO[0000] Creating new Go operator 'SnapScheduler'.
INFO[0000] Create cmd/manager/main.go
INFO[0000] Create build/Dockerfile
INFO[0000] Create build/bin/entrypoint
INFO[0000] Create build/bin/user_setup
INFO[0000] Create deploy/service_account.yaml
INFO[0000] Create deploy/role.yaml
INFO[0000] Create deploy/role_binding.yaml
INFO[0000] Create deploy/operator.yaml
INFO[0000] Create pkg/apis/apis.go
INFO[0000] Create pkg/controller/controller.go
INFO[0000] Create version/version.go
INFO[0000] Create .gitignore
INFO[0000] Create Gopkg.toml
INFO[0000] Run dep ensure ...
... <dep output> ...
INFO[0047] Run dep ensure done
INFO[0047] Project creation complete.

$ cd SnapScheduler
$

$ operator-sdk add api --api-version snapscheduler.backube/v1alpha1 --kind SnapshotPolicy
INFO[0000] Generating api version snapscheduler.backube/v1alpha1 for kind SnapshotPolicy.
INFO[0000] Create pkg/apis/snapscheduler/v1alpha1/snapshotpolicy_types.go
INFO[0000] Create pkg/apis/addtoscheme_snapscheduler_v1alpha1.go
INFO[0000] Create pkg/apis/snapscheduler/v1alpha1/register.go
INFO[0000] Create pkg/apis/snapscheduler/v1alpha1/doc.go
INFO[0000] Create deploy/crds/snapscheduler_v1alpha1_snapshotpolicy_cr.yaml
INFO[0001] Create deploy/crds/snapscheduler_v1alpha1_snapshotpolicy_crd.yaml
INFO[0006] Running deepcopy code-generation for Custom Resource group versions: [snapscheduler:[v1alpha1], ]
INFO[0008] Code-generation complete.
INFO[0010] Running OpenAPI code-generation for Custom Resource group versions: [snapscheduler:[v1alpha1], ]
INFO[0011] Create deploy/crds/snapscheduler_v1alpha1_snapshotpolicy_crd.yaml
INFO[0011] Code-generation complete.
INFO[0011] API generation complete.

$ operator-sdk add controller --api-version snapscheduler.backube/v1alpha1 --kind SnapshotPolicy
INFO[0000] Generating controller version snapscheduler.backube/v1alpha1 for kind SnapshotPolicy.
INFO[0000] Create pkg/controller/snapshotpolicy/snapshotpolicy_controller.go
INFO[0000] Create pkg/controller/add_snapshotpolicy.go
INFO[0000] Controller generation complete.
```

Copy the .git directory across

```
$ cp -a ../SnapScheduler_tmp/.git .
$
```

Ignore the vendor directory

```
$ echo /vendor/ >> .gitignore
$
```

Commit the new scaffolding

```
$ git add .
...

$ git commit -m 'Add scaffolding for sdk v0.5.0'
[sdk-scaffold c484ae0] Add scaffolding for sdk v0.5.0
 25 files changed, 1888 insertions(+)
 create mode 100644 .gitignore
 create mode 100644 Gopkg.lock
 create mode 100644 Gopkg.toml
 create mode 100644 build/Dockerfile
 create mode 100755 build/bin/entrypoint
 create mode 100755 build/bin/user_setup
 create mode 100644 cmd/manager/main.go
 create mode 100644 deploy/crds/snapscheduler_v1alpha1_snapshotpolicy_cr.yaml
 create mode 100644 deploy/crds/snapscheduler_v1alpha1_snapshotpolicy_crd.yaml
 create mode 100644 deploy/operator.yaml
 create mode 100644 deploy/role.yaml
 create mode 100644 deploy/role_binding.yaml
 create mode 100644 deploy/service_account.yaml
 create mode 100644 pkg/apis/addtoscheme_snapscheduler_v1alpha1.go
 create mode 100644 pkg/apis/apis.go
 create mode 100644 pkg/apis/snapscheduler/v1alpha1/doc.go
 create mode 100644 pkg/apis/snapscheduler/v1alpha1/register.go
 create mode 100644 pkg/apis/snapscheduler/v1alpha1/snapshotpolicy_types.go
 create mode 100644 pkg/apis/snapscheduler/v1alpha1/zz_generated.deepcopy.go
 create mode 100644 pkg/apis/snapscheduler/v1alpha1/zz_generated.defaults.go
 create mode 100644 pkg/apis/snapscheduler/v1alpha1/zz_generated.openapi.go
 create mode 100644 pkg/controller/add_snapshotpolicy.go
 create mode 100644 pkg/controller/controller.go
 create mode 100644 pkg/controller/snapshotschedule/snapshotpolicy_controller.go
 create mode 100644 version/version.go
```

Merge it back into master

```
$ git checkout master
Switched to branch 'master'
Your branch is up to date with 'origin/master'.
$ git merge sdk-scaffold
.....
```
