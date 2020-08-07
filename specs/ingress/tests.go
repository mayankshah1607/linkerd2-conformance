package ingress

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/linkerd/linkerd2-conformance/utils"
	"github.com/onsi/gomega"
)

func pingIP(ip string) error {
	req, err := http.NewRequest("GET", ip, nil)
	if err != nil {
		return err

	}

	req.Host = "example.com"

	client := http.Client{
		Timeout: 15 * time.Minute,
	}

	res, err := client.Do(req)
	if err != nil {
		return err

	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return fmt.Errorf("did not recieve status code 200. Recieved %d", res.StatusCode)

	}
	return nil

}

func getExternalIP(svc, ns string) (string, error) {
	h, _ := utils.GetHelperAndConfig()
	var ip string
	var err error

	err = h.RetryFor(time.Minute*5, func() error {
		ip, err = h.Kubectl("", "get", "svc", "-n", ns, svc, "-o", "jsonpath='{.status.loadBalancer.ingress[0].ip}'")
		if err != nil {
			return fmt.Errorf("failed to fetch external IP: %s", err.Error())
		}
		if strings.Trim(ip, "'") == "" {
			return fmt.Errorf("IP address is empty")
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return strings.Trim(ip, "'"), nil
}

func testIngress(tc testCase) {
	// install sample application
	utils.TestEmojivotoApp()
	utils.TestEmojivotoInject()

	h, _ := utils.GetHelperAndConfig()

	// install and inject controller
	for _, url := range tc.controllerURL {
		out, err := h.Kubectl("",
			"apply",
			"-f", url)
		gomega.Expect(err).Should(gomega.BeNil(),
			fmt.Sprintf("`kubectl apply` command failed: %s\n%s", out, utils.Err(err)))
	}

	err := h.CheckPods(tc.namespace, tc.controllerDeployName, 1)

	out, err := h.Kubectl("",
		"get", "-n", tc.namespace,
		"deploy", tc.controllerDeployName,
		"-o", "yaml")
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`kubectl get deploy` command failed: %s\n%s", out, utils.Err(err)))

	out, stderr, err := h.PipeToLinkerdRun(out, "inject", "-")
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`linkerd inject` command failed: %s\n%s", out, stderr))

	out, err = h.KubectlApply(out, tc.namespace)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`kubectl apply` command failed: %s\n%s", out, utils.Err(err)))

	err = h.CheckPods(tc.namespace, tc.controllerDeployName, 2)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to verify controller pods: %s", utils.Err(err)))

	err = utils.CheckProxyContainer(tc.controllerDeployName, tc.namespace)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("could not finx proxy container in controller deployment: %s", utils.Err(err)))

	// install ingress resource
	out, err = h.Kubectl("", "apply", "-f", tc.resourcePath)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`kubectl apply` command failed: %s\n%s", out, utils.Err(err)))

	extIP, err := getExternalIP(tc.lbSvcName, tc.namespace)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to fetch external IP: %s", utils.Err(err)))

	err = h.RetryFor(3*time.Minute, func() error {
		return pingIP(fmt.Sprintf("http://%s", strings.Trim(extIP, "'")))
	})
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to reach emojivoto: %s", utils.Err(err)))
}

func testClean(tc testCase) {
	// uninstall emojivoto
	utils.TestEmojivotoUninstall()
	h, _ := utils.GetHelperAndConfig()

	for _, url := range tc.controllerURL {
		out, err := h.Kubectl("", "delete",
			"--ignore-not-found",
			"-n", tc.namespace,
			"-f", url)
		gomega.Expect(err).Should(gomega.BeNil(),
			fmt.Sprintf("`kubectl delete` command failed: %s\n%s",
				out, utils.Err(err)))

	}

}
