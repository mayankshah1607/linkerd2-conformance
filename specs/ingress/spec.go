package ingress

import (
	"fmt"

	"github.com/linkerd/linkerd2-conformance/utils"
	"github.com/onsi/ginkgo"
)

type testCase struct {
	ingressName          string
	controllerURL        []string
	controllerDeployName string
	svcName              string
	namespace            string
	resourcePath         string
}

const (
	nginxURL         = "https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/static/provider/cloud/deploy.yaml"
	traefikURL       = "testdata/ingress/controllers/traefik.yaml"
	ambassadorCRDURL = "https://www.getambassador.io/yaml/aes-crds.yaml"
	ambassadorURL    = "https://www.getambassador.io/yaml/aes.yaml"
)

var testCases = []testCase{
	{
		ingressName:          "nginx",
		controllerURL:        []string{nginxURL},
		resourcePath:         "testdata/ingress/resources/nginx.yaml",
		controllerDeployName: "ingress-nginx-controller",
		svcName:              "ingress-nginx-controller",
		namespace:            "ingress-nginx",
	},
	{
		ingressName:          "traefik",
		controllerURL:        []string{traefikURL},
		resourcePath:         "testdata/ingress/resources/traefik.yaml",
		controllerDeployName: "traefik-ingress-controller",
		namespace:            "kube-system",
		svcName:              "traefik-ingress-controller",
	},
	{
		ingressName:          "ambassador",
		controllerURL:        []string{ambassadorCRDURL, ambassadorURL},
		resourcePath:         "testdata/ingress/resources/ambassador.yaml",
		controllerDeployName: "ambassador",
		namespace:            "ambassador",
		svcName:              "ambassador",
	},
}

// RunIngressTests runs the specs for ingress
func RunIngressTests() bool {
	return ginkgo.Describe("ingress: ", func() {
		_, c := utils.GetHelperAndConfig()

		_ = utils.ShouldTestSkip(c.SkipIngress(), "Skipping ingress tests")

		for _, tc := range testCases {
			tc := tc //pin
			if c.ShouldTestIngressOfType(tc.ingressName) {
				ginkgo.Context(fmt.Sprintf("%s:", tc.ingressName), func() {
					ginkgo.It("should work with Linkerd", func() {
						testIngress(tc)
					})

					if c.ShouldCleanIngressInstallation(tc.ingressName) {
						ginkgo.It("should delete resources created for testing", func() {
							testClean(tc)
						})
					}
				})
			}
		}
	})
}
