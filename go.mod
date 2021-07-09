module github.com/backube/snapscheduler

go 1.16

require (
	github.com/go-logr/logr v0.3.0
	github.com/kubernetes-csi/external-snapshotter v1.2.2
	github.com/kubernetes-csi/external-snapshotter/v2 v2.1.5
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/openshift/custom-resource-status v1.1.0
	github.com/robfig/cron/v3 v3.0.1
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.3
)
