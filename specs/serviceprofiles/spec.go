package serviceprofiles

import (
	"github.com/linkerd/linkerd2-conformance/utils"
	"github.com/onsi/ginkgo"
)

// RunSpTests runs tests for retries and timeouts
func RunSpTests() bool {
	return ginkgo.Describe("service profiles:", func() {
		_, c := utils.GetHelperAndConfig()
		_ = utils.ShouldTestSkip(c.SkipSP(), "Skipping tests related to ServiceProfiles")

		ginkgo.It("can install sample application [booksapp]", testSampleApp)
		ginkgo.It("can install service profiles for booksapp", testInstallSP)
		ginkgo.It("can get expected routes", testRoutes)
		ginkgo.It("retries can be configured correctly", testRetries)
		// ginkgo.It("timeouts can be configured correctly", testTimeouts)

		if c.CleanSP() {
			ginkgo.It("must clean up resources created for testing", testClean)
		}
	})
}
