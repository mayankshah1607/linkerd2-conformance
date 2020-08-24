package tap

import (
	"fmt"
	"strings"

	"github.com/linkerd/linkerd2-conformance/utils"
	"github.com/linkerd/linkerd2/testutil"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

var (
	expectedT1 = testutil.TapEvent{
		Method:     "POST",
		Authority:  "t1-svc:9090",
		Path:       "/buoyantio.bb.TheService/theFunction",
		HTTPStatus: "200",
		GrpcStatus: "OK",
		TLS:        "true",
		LineCount:  3,
	}

	expectedT2 = testutil.TapEvent{
		Method:     "POST",
		Authority:  "t2-svc:9090",
		Path:       "/buoyantio.bb.TheService/theFunction",
		HTTPStatus: "200",
		GrpcStatus: "Unknown",
		TLS:        "true",
		LineCount:  3,
	}

	expectedT3 = testutil.TapEvent{
		Method:     "POST",
		Authority:  "t3-svc:8080",
		Path:       "/",
		HTTPStatus: "200",
		GrpcStatus: "",
		TLS:        "true",
		LineCount:  3,
	}

	expectedGateway = testutil.TapEvent{
		Method:     "GET",
		Authority:  "gateway-svc:8080",
		Path:       "/",
		HTTPStatus: "500",
		GrpcStatus: "",
		TLS:        "true",
		LineCount:  3,
	}

	prefixedNs = ""
)

func testTapAppDeploy() {
	h, _ := utils.GetHelperAndConfig()
	out, stderr, err := h.LinkerdRun("inject", "--manual", "testdata/tap/tap_application.yaml")
	gomega.Expect(err).Should(gomega.BeNil(), "`inject` command failed: %s\n%s", out, stderr)

	prefixedNs = h.GetTestNamespace("tap-test")
	err = h.CreateDataPlaneNamespaceIfNotExists(prefixedNs, nil)
	gomega.Expect(err).Should(gomega.BeNil(), "failed to create namespace \"%s\": %s", prefixedNs, utils.Err(err))

	out, err = h.KubectlApply(out, prefixedNs)
	gomega.Expect(err).Should(gomega.BeNil(), "`kubectl apply` command failed: %s\n%s", out, utils.Err(err))

	for _, deploy := range []string{"t1", "t2", "t3", "gateway"} {
		if err := h.CheckPods(prefixedNs, deploy, 1); err != nil {
			if _, ok := err.(*testutil.RestartCountError); !ok {
				ginkgo.Fail(fmt.Sprintf("failed to verify pod count in deploy \"%s\": %s", deploy, err.Error()))
			}
		}

		err := h.CheckDeployment(prefixedNs, deploy, 1)
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("error validating deployment \"%s\": %s", deploy, utils.Err(err)))
	}
}

func testTapDeploy() {
	h, _ := utils.GetHelperAndConfig()
	events, err := testutil.Tap("deploy/t1", h, "--namespace", prefixedNs)
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("tap failed: %s", utils.Err(err)))

	err = testutil.ValidateExpected(events, expectedT1)
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to validate tap: %s", utils.Err(err)))
}

func testTapDisabledDeploy() {
	h, _ := utils.GetHelperAndConfig()

	out, stderr, err := h.LinkerdRun("tap", "deploy/t4", "--namespace", prefixedNs)

	gomega.Expect(out).Should(gomega.Equal(""), fmt.Sprintf("unexpected output: %s", out))
	gomega.Expect(err).ShouldNot(gomega.BeNil(), "expected error, got none")
	gomega.Expect(stderr).ShouldNot(gomega.Equal(""), "expected error, got none")

	expectedErr := "Error: all pods found for deployment/t4 have tapping disabled"
	errs := strings.Split(stderr, "\n")
	gomega.Expect(errs[0]).Should(gomega.Equal(expectedErr), fmt.Sprintf("expected [%s], got [%s]", expectedErr, errs[0]))
}

func testTapSvcCall() {
	h, _ := utils.GetHelperAndConfig()

	events, err := testutil.Tap("deploy/gateway", h, "--to", "svc/t2-svc", "--namespace", prefixedNs)
	gomega.Expect(err).Should(gomega.BeNil(), "failed to tap service call")

	err = testutil.ValidateExpected(events, expectedT2)
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to validate service call tap: %s", utils.Err(err)))
}

func testTapPod() {
	h, _ := utils.GetHelperAndConfig()
	deploy := "t3"
	pods, err := h.GetPodNamesForDeployment(prefixedNs, deploy)
	gomega.Expect(err).Should(gomega.BeNil(), "failed to get pods for deploy/t3")

	gomega.Expect(len(pods)).Should(gomega.Equal(1), "expected exactly one pod")

	events, err := testutil.Tap("pod/"+pods[0], h, "--namespace", prefixedNs)
	gomega.Expect(err).Should(gomega.BeNil(), "failed to tap pod")

	err = testutil.ValidateExpected(events, expectedT3)
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to validate tap: %s", utils.Err(err)))
}

func testTapFilterMethod() {
	h, _ := utils.GetHelperAndConfig()

	events, err := testutil.Tap("deploy/gateway", h, "--namespace", prefixedNs, "--method", "GET")
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("error filtering events by method: %s", utils.Err(err)))

	err = testutil.ValidateExpected(events, expectedGateway)
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("error validating filtered tap events by method: %s", utils.Err(err)))
}

func testTapFilterAuthority() {
	h, _ := utils.GetHelperAndConfig()

	events, err := testutil.Tap("deploy/gateway", h, "--namespace", prefixedNs, "--authority", "t1-svc:9090")
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("error filtering events by authority: %s", utils.Err(err)))

	err = testutil.ValidateExpected(events, expectedT1)
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("error validating filtered tap events by authority: %s", utils.Err(err)))
}

func testDeleteTapApp() {
	h, _ := utils.GetHelperAndConfig()

	out, err := h.Kubectl("", "delete", "-n", prefixedNs, "-f", "testdata/tap/tap_application.yaml")
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to delete application: %s\n%s", out, err))

	out, err = h.Kubectl("", "delete", "ns", prefixedNs)
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to delete namespace \"%s\": %s\n%s", prefixedNs, out, err))
}
