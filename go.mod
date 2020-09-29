module github.com/backube/snapscheduler

go 1.13

require (
	github.com/go-logr/logr v0.1.0
	github.com/kubernetes-csi/external-snapshotter v1.2.2
	github.com/kubernetes-csi/external-snapshotter/v2 v2.1.1
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.1
	github.com/openshift/custom-resource-status v0.0.0-20200602122900-c002fd1547ca
	github.com/robfig/cron/v3 v3.0.1
	github.com/stretchr/testify v1.5.1 // indirect
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.6
	sigs.k8s.io/controller-runtime v0.6.2
)
