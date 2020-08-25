package ingress

import (
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/linkerd/linkerd2-conformance/utils"
	"github.com/onsi/gomega"
)

func httpGet(url string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err

	}

	req.Close = true
	req.Host = "example.com"

	client := http.Client{
		Timeout:   15 * time.Minute,
		Transport: &http.Transport{MaxConnsPerHost: 20},
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

func beginPortForward(ns, obj string) (*exec.Cmd, error) {
	cmd := exec.Command("kubectl",
		"-n", ns,
		"port-forward",
		obj,
		"8080:80")

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

func stopPortforward(cmd *exec.Cmd) error {
	if err := cmd.Process.Kill(); err != nil {
		return err
	}
	return nil
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

	// CheckDeployment retries only for 30s
	// This wrapper will ensure that CheckDeployment
	// is tried for longer
	err := h.RetryFor(5*time.Minute, func() error {
		err := h.CheckDeployment(tc.namespace, tc.controllerDeployName, 1)
		if err != nil {
			return err
		}
		return nil
	})

	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("CheckDeployment timed-out: %s", utils.Err(err)))

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

	err = h.CheckDeployment(tc.namespace, tc.controllerDeployName, 1)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to verify controller pods: %s", utils.Err(err)))

	err = utils.CheckProxyContainer(tc.controllerDeployName, tc.namespace)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("could not finx proxy container in controller deployment: %s", utils.Err(err)))

	// install ingress resource
	out, err = h.Kubectl("", "apply", "-f", tc.resourcePath)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`kubectl apply` command failed: %s\n%s", out, utils.Err(err)))

	var cmd *exec.Cmd
	if tc.ingressName != "ambassador" {
		cmd, err = beginPortForward(tc.namespace, "svc/"+tc.svcName)
	} else {
		cmd, err = beginPortForward("emojivoto", "svc/web-ambassador")
	}

	defer func() {
		_ = stopPortforward(cmd)
	}()

	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`kubectl port-forward` command failed: %s", utils.Err(err)))

	url := "http://127.0.0.1:8080"
	err = h.RetryFor(3*time.Minute, func() error {
		return httpGet(url)
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
