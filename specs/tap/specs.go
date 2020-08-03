package tap

import (
	"github.com/linkerd/linkerd2-conformance/utils"
	"github.com/onsi/ginkgo"
)

func RunTapTests() bool {
	return ginkgo.Describe("tap:", func() {
		_, c := utils.GetHelperAndConfig()

		_ = utils.ShouldTestSkip(c.SkipTap(), "Skipping tap tests")

		ginkgo.It("creating sample application", testTapAppDeploy)
		ginkgo.It("can tap a deployment", testTapDeploy)
		ginkgo.It("cannot tap a disabled deployment", testTapDisabledDeploy)
		ginkgo.It("can tap a service call", testTapSvcCall)
		ginkgo.It("can tap a pod", testTapPod)
		ginkgo.It("can filter tap events by method", testTapFilterMethod)
		ginkgo.It("can filter tap events by authority", testTapFilterAuthority)

		if c.CleanTap() {
			ginkgo.It("deleting sample application", testDeleteTapApp)
		}
	})
}
