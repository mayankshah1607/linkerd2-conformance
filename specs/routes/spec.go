package routes

import "github.com/onsi/ginkgo"

func RunRoutesTests() bool {
	return ginkgo.Describe("routes:", func() {
		ginkgo.It("installing smoke-test application", testInstallSmokeTest)
		ginkgo.It("installing ServiceProfiles for smoke-test", testInstallSPSmokeTest)
		ginkgo.It("installing ServiceProfiles for control plane", testInstallSPContolPlane)
		ginkgo.It("running `linkerd routes`", testRoutes)
		ginkgo.It("uninstalling smoke-test", testUninstallSmokeTest)
		ginkgo.It("uninstalling control plane ServiceProfiles", testUninstallControlPlaneServiceProfile)
	})
}
