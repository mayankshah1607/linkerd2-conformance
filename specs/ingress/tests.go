package ingress

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/linkerd/linkerd2-conformance/utils"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

var (
	nginx   = "nginx"
	traefik = "traefik"

	nginxNs   = "ingress-nginx"
	traefikNs = "kube-system"

	nginxController   = "ingress-nginx-controller"
	traefikController = "traefik-ingress-controller"
)

func pingEmojivoto(ip string) error {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s", ip), nil)
	if err != nil {
		return err
	}

	req.Host = "example.com"

	client := http.Client{
		Timeout: 3 * time.Minute,
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

func testIngress(ingressName, deploy, ns, controllerYAMLPath, resourceYAMLPath string) {
	h, _ := utils.GetHelperAndConfig()

	utils.TestEmojivotoApp()
	utils.TestEmojivotoInject()

	ginkgo.By(fmt.Sprintf("Creating %s controller", ingressName))
	_, err := h.Kubectl("", "apply", "-f", controllerYAMLPath)

	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to create %s controller: %s", ingressName, utils.Err(err)))

	err = h.CheckPods(ns, deploy, 1)
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to verify %s controller pods: %s", ingressName, utils.Err(err)))

	ginkgo.By(fmt.Sprintf("Injecting linkerd into %s ingress controller pods", ingressName))
	out, err := h.Kubectl("", "get", "-n", ns, "deploy", deploy, "-o", "yaml")
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to get YAML manifest for deploy/%s: %s", deploy, utils.Err(err)))

	out, stderr, err := h.PipeToLinkerdRun(out, "inject", "-")
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to inject: %s", stderr))

	_, err = h.KubectlApply(out, ns)
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to apply injected manifests: %s", utils.Err(err)))

	err = h.CheckPods(ns, deploy, 1)
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to verify %s controller pods: %s", ingressName, utils.Err(err)))

	ginkgo.By(fmt.Sprintf("Verifying if %s ingress controller pods have been injected", ingressName))

	// Wait upto 3mins for proxy container to show up
	err = utils.CheckProxyContainer(deploy, ns)
	gomega.Expect(err).Should(gomega.BeNil(), utils.Err(err))

	ginkgo.By("Applying ingress resource")
	_, err = h.Kubectl("", "apply", "-f", resourceYAMLPath)
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to create %s ingress resource: %s", ingressName, utils.Err(err)))

	ginkgo.By("Checking if emojivoto is reachable")
	ip, err := getExternalIP(deploy, ns)
	gomega.Expect(err).Should(gomega.BeNil(), utils.Err(err))

	err = h.RetryFor(3*time.Minute, func() error {
		return pingEmojivoto(strings.Trim(ip, "'"))
	})

	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to reach emojivoto: %s", utils.Err(err)))

	ginkgo.By(fmt.Sprintf("Removing %s ingress controller", ns))

	_, err = h.Kubectl("", "delete", "-f", controllerYAMLPath)
	gomega.Expect(err).Should(gomega.BeNil(), utils.Err(err))

	ginkgo.By("Uninstalling emojivoto")
	utils.TestEmojivotoUninstall()
}

func testNginx() {
	var (
		nginxControllerYAML = "testdata/ingress/controllers/nginx.yaml"
		nginxResourceYAML   = "testdata/ingress/resources/nginx.yaml"
	)
	testIngress(nginx, nginxController, nginxNs, nginxControllerYAML, nginxResourceYAML)
}

func testTraefik() {
	var (
		traefikControllerYAML = "testdata/ingress/controllers/traefik.yaml"
		traefikResourceYAML   = "testdata/ingress/resources/traefik.yaml"
	)
	testIngress(traefik, traefikController, traefikNs, traefikControllerYAML, traefikResourceYAML)
}
