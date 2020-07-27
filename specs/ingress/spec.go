package ingress

import (
	"fmt"

	"github.com/linkerd/linkerd2-conformance/utils"
	"github.com/onsi/ginkgo"
)

var (
	nginx      = "nginx"
	traefik    = "traefik"
	ambassador = "ambassador"
	gloo       = "gloo"
	contour    = "contour"

	nginxNs      = "ingress-nginx"
	traefikNs    = "kube-system"
	ambassadorNs = "ingress-ambassador"
	glooNs       = "gloo-system"
	contourNs    = "projectcontour"
	kuardNs      = "default"

	nginxController      = "ingress-nginx-controller"
	traefikController    = "traefik-ingress-controller"
	ambassadorController = "ambassador"
	contourController    = "contour"
	kuard                = "kuard"

	kuardYAML = "https://projectcontour.io/examples/kuard.yaml"
)

type testRunner func()

var testCases = []struct {
	ingressName          string
	controllerYAML       string
	controllerDeployName string
	controllerNs         string
	testRunner           testRunner
}{
	{
		ingressName:          nginx,
		controllerYAML:       "testdata/ingress/controllers/nginx.yaml",
		controllerDeployName: nginxController,
		controllerNs:         nginxNs,
		testRunner:           testNginx,
	},
	{
		ingressName:          traefik,
		controllerYAML:       "testdata/ingress/controllers/traefik.yaml",
		controllerDeployName: traefikController,
		controllerNs:         traefikNs,
		testRunner:           testTraefik,
	},
	{
		ingressName:          ambassador,
		controllerYAML:       "testdata/ingress/controllers/ambassador.yaml",
		controllerDeployName: ambassadorController,
		controllerNs:         ambassadorNs,
		testRunner:           testAmbassador,
	},
	{
		ingressName:          gloo,
		controllerYAML:       "", // `glooctl` is used for installation
		controllerDeployName: "", // `glooctl` handles the checks
		controllerNs:         glooNs,
		testRunner:           testGloo,
	},
	{
		ingressName:          contour,
		controllerYAML:       "testdata/ingress/controllers/contour.yaml",
		controllerDeployName: contourController,
		controllerNs:         contourNs,
		testRunner:           testContour,
	},
}

// RunIngressTests runs the specs for ingress
func RunIngressTests() bool {
	return ginkgo.Describe("ingress", func() {
		_, c := utils.GetHelperAndConfig()

		_ = utils.ShouldTestSkip(c.SkipIngress(), "Skipping ingress tests")

		for _, tc := range testCases {
			tc := tc //pin
			if c.ShouldTestIngressOfType(tc.ingressName) {
				ginkgo.Context(fmt.Sprintf("%s:", tc.ingressName), func() {
					ginkgo.It("sample application can be installed", func() {
						testSampleAppInstall(tc.ingressName)
					})

					ginkgo.It(fmt.Sprintf("%s ingress controller must work with Linkerd", tc.ingressName), func() {
						if c.ShouldInstallIngressOfType(tc.ingressName) {
							testControllerInstall(tc.ingressName,
								tc.controllerYAML,
								tc.controllerDeployName,
								tc.controllerNs)
						}

						tc.testRunner() // run main body of the test
					})

					if c.ShouldCleanIngressInstallation(tc.ingressName) {

						ginkgo.It("sample application can be uninstalled", func() {
							testSampleAppUninstall(tc.ingressName)
						})

						ginkgo.It("controller can be uninstalled", func() {
							testControllerUninstall(tc.ingressName, tc.controllerYAML, tc.controllerNs)
						})
					}
				})
			}
		}
	})
}
