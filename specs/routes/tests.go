package routes

import (
	"fmt"
	"strings"

	"github.com/linkerd/linkerd2-conformance/utils"
	"github.com/linkerd/linkerd2/testutil"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

var ns = "smoke-test"

func testInstallSmokeTest() {
	h, _ := utils.GetHelperAndConfig()

	cmd := []string{"inject", "testdata/routes/smoke_test.yaml"}
	out, stderr, err := h.LinkerdRun(cmd...)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to inject manifests: %s\n%s", utils.Err(err), stderr))

	prefixedNs := h.GetTestNamespace(ns)
	err = h.CreateDataPlaneNamespaceIfNotExists(prefixedNs, map[string]string{})
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to create namespace: %s", utils.Err(err)))

	out, err = h.KubectlApply(out, prefixedNs)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to create resources %s\n%s", utils.Err(err), out))

	for _, deploy := range []string{"smoke-test-terminus", "smoke-test-gateway"} {
		if err := h.CheckPods(prefixedNs, deploy, 1); err != nil {
			if _, ok := err.(*testutil.RestartCountError); !ok {
				ginkgo.Fail(fmt.Sprintf("CheckPods timed-out: %s", err.Error()))
			}
		}
	}

	url, err := h.URLFor(prefixedNs, "smoke-test-gateway", 8080)
	gomega.Expect(err).Should(gomega.BeNil(),
		"failed to get URL for [smoke-test-gateway]")
	output, err := h.HTTPGetURL(url)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to reach smoke-test-gateway: %s\n%s", utils.Err(err), output))

	expectedStringPayload := "\"payload\":\"BANANA\""
	gomega.Expect(output).Should(gomega.ContainSubstring(expectedStringPayload),
		"output does not contain expected substring")
}

func testInstallSPSmokeTest() {
	h, _ := utils.GetHelperAndConfig()
	prefixedNs := h.GetTestNamespace(ns)

	bbProto, err := testutil.ReadFile("testdata/routes/api.proto")
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to read proto file: %s", utils.Err(err)))

	cmd := []string{"profile", "-n", prefixedNs, "--proto", "-", "smoke-test-terminus-svc"}
	bbSP, stderr, err := h.PipeToLinkerdRun(bbProto, cmd...)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to produce ServiceProfiles: %s\n%s", utils.Err(err), stderr))

	out, err := h.KubectlApply(bbSP, prefixedNs)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to install ServiceProfiles: %s\n%s", utils.Err(err), out))
}

func testInstallSPContolPlane() {
	h, _ := utils.GetHelperAndConfig()

	cmd := []string{"install-sp"}

	out, stderr, err := h.LinkerdRun(cmd...)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to generate ServiceProfiles: %s\n%s", utils.Err(err), stderr))

	out, err = h.KubectlApply(out, h.GetLinkerdNamespace())
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to install ServiceProfiles: %s\n%s", utils.Err(err), out))
}

func testRoutes() {
	h, _ := utils.GetHelperAndConfig()

	ginkgo.By("Testing control plane routes")

	cmd := []string{"routes", "--namespace", h.GetLinkerdNamespace(), "deploy"}
	out, stderr, err := h.LinkerdRun(cmd...)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`linkerd routes` command failed: %s\n%s", out, stderr))

	routeStrings := []struct {
		s string
		c int
	}{
		{"linkerd-controller-api", 7},
		{"linkerd-destination", 1},
		{"linkerd-dst", 3},
		{"linkerd-grafana", 13},
		{"linkerd-identity", 2},
		{"linkerd-prometheus", 5},
		{"linkerd-web", 2},

		{"POST /api/v1/ListPods", 1},
		{"POST /api/v1/", 7},
		{"POST /io.linkerd.proxy.destination.Destination/Get", 2},
		{"GET /api/annotations", 1},
		{"GET /api/", 9},
		{"GET /public/", 3},
		{"GET /api/v1/", 2},
	}

	for _, r := range routeStrings {
		count := strings.Count(out, r.s)
		gomega.Expect(count).Should(gomega.Equal(r.c),
			fmt.Sprintf("expected %d occurences of %s, got %d", r.c, r.s, count))
	}

	ginkgo.By("Testing smoke-test routes")
	prefixedNs := h.GetTestNamespace(ns)
	cmd = []string{"routes", "--namespace", prefixedNs, "deploy"}
	golden := "routes/routes.smoke.golden"

	out, stderr, err = h.LinkerdRun(cmd...)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`linkerd routes` command failed: %s\n%s", out, stderr))

	err = h.ValidateOutput(out, golden)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to validate output: %s", utils.Err(err)))
}

func testUninstallSmokeTest() {
	h, _ := utils.GetHelperAndConfig()
	prefixedNs := h.GetTestNamespace(ns)

	out, err := h.Kubectl("", "delete", "ns", prefixedNs)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`kubectl delete` command failed: %s\n%s", utils.Err(err), out))

}

func testUninstallControlPlaneServiceProfile() {
	h, _ := utils.GetHelperAndConfig()

	cmd := []string{"install-sp"}

	out, stderr, err := h.LinkerdRun(cmd...)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to generate ServiceProfiles: %s\n%s", utils.Err(err), stderr))

	out, err = h.Kubectl(out, "-n", h.GetLinkerdNamespace(), "delete", "-f", "-")
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to remove ServiceProfiles: %s\n%s", utils.Err(err), out))
}
