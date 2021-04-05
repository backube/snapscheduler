/*
Copyright (C) 2019  The snapscheduler authors

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package e2e

import (
	"os"
	"testing"

	snapv1alpha1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	snapv1beta1 "github.com/kubernetes-csi/external-snapshotter/v2/pkg/apis/volumesnapshot/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	snapschedulerv1 "github.com/backube/snapscheduler/api/v1"
	sscontroller "github.com/backube/snapscheduler/controllers"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"E2E Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter)))

	By("initializing client")
	cfg = ctrl.GetConfigOrDie()
	Expect(cfg).ToNot(BeNil())

	err := snapschedulerv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	// We need access to VolumeSnapshot objects
	err = snapv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = snapv1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	// Initialize MVSnapshot so we can work w/ both alpha and beta snaps
	err = sscontroller.VersionChecker.SetConfig(cfg)
	Expect(err).NotTo(HaveOccurred())
	err = sscontroller.VersionChecker.Refresh(logf.Log.Logger)
	Expect(err).NotTo(HaveOccurred())

	// Allow override of StorageClass and SnapshotClass names via environment
	// variables
	if v := os.Getenv(EnvStorageClass); v != "" {
		storageClassName = v
	}
	logf.Log.Info("using StorageClassName", "sc", storageClassName)
	if v := os.Getenv(EnvSnapClass); v != "" {
		snapshotClassName = v
	}
	logf.Log.Info("using SnapshotClassName", "snapclass", snapshotClassName)

	close(done)
}, 60)

// var _ = AfterSuite(func() {
// 	By("tearing down the test environment")
// 	err := testEnv.Stop()
// 	Expect(err).ToNot(HaveOccurred())
// })
