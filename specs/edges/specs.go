package edges

import (
	"github.com/linkerd/linkerd2-conformance/utils"
	"github.com/onsi/ginkgo"
)

// RunEdgesTests runs the specs for `linkerd edges`
func RunEdgesTests() bool {
	return ginkgo.Describe("`linkerd edges`: ", func() {
		_, c := utils.GetHelperAndConfig()
		_ = utils.ShouldTestSkip(c.SkipEdges(), "Skipping `linked edges` tests")

		ginkgo.It("can deploy terminus", testDeployTerminus)
		ginkgo.It("can deploy slow-cooker", testDeploySlowCooker)
		ginkgo.It("can get the registered edges", testEdges)

		if c.CleanEdges() {
			ginkgo.It("can delete resources created for testing", testClean)
		}
	})
}
