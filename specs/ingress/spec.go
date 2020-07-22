package ingress

import (
	"fmt"

	"github.com/linkerd/linkerd2-conformance/utils"
	"github.com/onsi/ginkgo"
)

func specMessage(controllerName string) string {
	return fmt.Sprintf("can work with %s ingress controller", controllerName)
}

// RunIngressTests runs the specs for ingress
func RunIngressTests() bool {
	return ginkgo.Describe("ingress: ", func() {
		_, c := utils.GetHelperAndConfig()

		_ = utils.ShouldTestSkip(c.SkipIngress(), "Skipping ingress tests")

		if c.ShouldTestIngressOfType(nginx) {
			ginkgo.It(specMessage(nginx), testNginx)
		}

		if c.ShouldTestIngressOfType(traefik) {
			ginkgo.It(specMessage(traefik), testTraefik)
		}

		ginkgo.It("can uninstall emojivoto app", utils.TestEmojivotoUninstall)
	})
}
