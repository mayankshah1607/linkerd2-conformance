package ingress

import (
	"fmt"

	"github.com/linkerd/linkerd2-conformance/utils"
	"github.com/onsi/ginkgo"
)

type testRunner func()

var testCases = []struct {
	ingressType string
	testRunner  testRunner
}{
	{
		ingressType: nginx,
		testRunner:  testNginx,
	},
	{
		ingressType: traefik,
		testRunner:  testTraefik,
	},
	{
		ingressType: ambassador,
		testRunner:  testAmbassador,
	},
	{
		ingressType: gloo,
		testRunner:  testGloo,
	},
}

func specMessage(controllerName string) string {
	return fmt.Sprintf("can work with %s ingress controller", controllerName)
}

// RunIngressTests runs the specs for ingress
func RunIngressTests() bool {
	return ginkgo.Describe("ingress: ", func() {
		_, c := utils.GetHelperAndConfig()

		_ = utils.ShouldTestSkip(c.SkipIngress(), "Skipping ingress tests")

		for _, tc := range testCases {
			tc := tc //pin
			if c.ShouldTestIngressOfType(tc.ingressType) {
				ginkgo.It(specMessage(tc.ingressType), func() {
					tc.testRunner()
				})
			}
		}
	})
}
