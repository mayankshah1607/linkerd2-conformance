package edges

import "github.com/onsi/ginkgo"

func RunEdgesTests() bool {
	return ginkgo.Describe("`linkerd edges`: ", func() {
		ginkgo.It("can deploy terminus", testDeployTerminus)
		ginkgo.It("can deploy slow-cooker", testDeploySlowCooker)
		ginkgo.It("can get the registered edges", testEdges)
	})
}
