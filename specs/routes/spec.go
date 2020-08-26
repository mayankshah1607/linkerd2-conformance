package routes

import (
	"github.com/linkerd/linkerd2-conformance/utils"
	"github.com/onsi/ginkgo"
)

func RunRoutesTests() bool {
	return ginkgo.Describe("routes:", func() {
		_, c := utils.GetHelperAndConfig()
		_ = utils.ShouldTestSkip(c.SkipRoutes(), "Skipping routes test")

		ginkgo.It("installing smoke-test application", testInstallSmokeTest)
		ginkgo.It("installing ServiceProfiles for smoke-test", testInstallSPSmokeTest)
		ginkgo.It("installing ServiceProfiles for control plane", testInstallSPContolPlane)
		ginkgo.It("running `linkerd routes`", testRoutes)

		if c.CleanRoutes() {
			ginkgo.It("uninstalling smoke-test", testUninstallSmokeTest)
			ginkgo.It("uninstalling control plane ServiceProfiles", testUninstallControlPlaneServiceProfile)
		}
	})
}
