package edges

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/linkerd/linkerd2-conformance/utils"
	"github.com/linkerd/linkerd2/testutil"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

var (
	testNs        string
	terminuspodIP string
)

func testDeployTerminus() {
	h, _ := utils.GetHelperAndConfig()

	testNs = h.GetTestNamespace("direct-edges-test")
	err := h.CreateControlPlaneNamespaceIfNotExists(testNs)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to create namespace '%s': %s", testNs, utils.Err(err)))

	out, stderr, err := h.LinkerdRun("inject", "--manual", "testdata/edges/terminus.yaml")
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to inject terminus: %s\n%s", out, stderr))

	out, err = h.KubectlApply(out, testNs)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`kubectl apply` command failed: %s\n%s", out, utils.Err(err)))

	if err := h.CheckPods(testNs, "terminus", 1); err != nil {
		if _, ok := err.(*testutil.RestartCountError); !ok {
			ginkgo.Fail(fmt.Sprintf("CheckPods timed-out: %s", err))
		}
	}

	err = h.CheckDeployment(testNs, "terminus", 1)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("CheckDeployment timed-out: %s", utils.Err(err)))

	terminuspodIP, err = h.Kubectl("", "-n", testNs,
		"get", "pod",
		"-ojsonpath=\"{.items[*].status.podIP}\"")
	gomega.Expect(err).Should(gomega.BeNil(),
		"failed to get pod ip: `kubectl get pod command failed` - %s", utils.Err(err))

	terminuspodIP = strings.Trim(terminuspodIP, "\"")
}

func testDeploySlowCooker() {
	h, _ := utils.GetHelperAndConfig()
	b, err := ioutil.ReadFile("testdata/edges/slow-cooker.yaml")
	gomega.Expect(err).Should(gomega.BeNil(),
		"error reading file slow-cooker.yaml")

	slowcooker := string(b)
	slowcooker = strings.ReplaceAll(slowcooker, "___TERMINUS_POD_IP___", terminuspodIP)

	out, stderr, err := h.PipeToLinkerdRun(slowcooker, "inject", "--manual", "-")
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`linkerd inject` command failed: %s\n%s", out, stderr))

	out, err = h.KubectlApply(out, testNs)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`kubectl apply` command failed: %s\n%s", out, err))

	if err := h.CheckPods(testNs, "slow-cooker", 1); err != nil {
		if _, ok := err.(*testutil.RestartCountError); !ok {
			ginkgo.Fail(fmt.Sprintf("CheckPods timed-out: %s", err))
		}
	}

	err = h.CheckDeployment(testNs, "slow-cooker", 1)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("CheckDeployment timed-out: %s", utils.Err(err)))
}

func testEdges() {
	h, _ := utils.GetHelperAndConfig()

	timeout := 50 * time.Second
	err := h.RetryFor(timeout, func() error {
		out, stderr, err := h.LinkerdRun("-n", testNs, "-o", "json", "edges", "deploy")
		gomega.Expect(err).Should(gomega.BeNil(),
			fmt.Sprintf("`linkerd edges` command failed: %s\n%s", out, stderr))

		tpl := template.Must(template.ParseFiles("testdata/edges/direct_edges.golden"))
		vars := struct {
			Ns        string
			ControlNs string
		}{
			testNs,
			h.GetLinkerdNamespace(),
		}
		var buf bytes.Buffer

		if err := tpl.Execute(&buf, vars); err != nil {
			return fmt.Errorf("failed to parse direct_edges.golden template: %s", err)
		}

		r := regexp.MustCompile(buf.String())
		if !r.MatchString(out) {
			return fmt.Errorf("expected output:\n%s\nactual output:\n%s", buf.String(), out)
		}
		return nil
	})
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to verify edges:\n%s", err))
}

func testClean() {
	h, _ := utils.GetHelperAndConfig()
	out, err := h.Kubectl("", "delete", "ns", testNs)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`kubectl delete ns` command failed: %s\n%s", out, err))
}
