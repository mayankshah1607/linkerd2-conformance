package specs

import (
	"testing"

	"github.com/linkerd/linkerd2-conformance/specs/ingress"
	"github.com/linkerd/linkerd2-conformance/specs/inject"
	"github.com/linkerd/linkerd2-conformance/specs/lifecycle"
	"github.com/linkerd/linkerd2-conformance/specs/stat"
	"github.com/linkerd/linkerd2-conformance/specs/tap"
	"github.com/linkerd/linkerd2-conformance/utils"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

func runBeforeAndAfterEachSetup() {
	h, c := utils.GetHelperAndConfig()
	if !c.SingleControlPlane() {
		_ = ginkgo.BeforeEach(func() {
			utils.InstallLinkerdControlPlane(h, c)
		})

		_ = ginkgo.AfterEach(func() {
			utils.UninstallLinkerdControlPlane(h)
		})
	}
}

// runLifecycleTests needs to have be declared separately in its own
// Describe block so that it does not interfere with the BeforeEach and AfterEach blocls
// of the main tests
func runLifecycleTests() bool {
	return ginkgo.Describe("", func() {
		_ = lifecycle.RunLifecycleTest()
	})
}

func runPrimaryTests() bool {
	h, c := utils.GetHelperAndConfig()
	return ginkgo.Describe("", func() {
		runBeforeAndAfterEachSetup()

		// add primary tests here
		_ = inject.RunInjectTests()
		_ = tap.RunTapTests()
		_ = ingress.RunIngressTests()
		_ = stat.RunStatTests()

		// a separate check for running uninstall must always occur at the end
		if c.SingleControlPlane() && h.Uninstall() {
			_ = lifecycle.RunUninstallTest()
		}
	})
}

func runConformanceTestsCallback() {
	_ = runLifecycleTests()
	_ = runPrimaryTests()
}

func RunConformanceTests(t *testing.T) {
	_ = ginkgo.Describe("", runConformanceTestsCallback)

	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Linkerd2 conformance tests")
}
