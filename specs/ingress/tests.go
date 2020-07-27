package ingress

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/linkerd/linkerd2-conformance/utils"
	"github.com/onsi/ginkgo"
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

func testSampleAppInstall(ingressName string) {
	h, _ := utils.GetHelperAndConfig()

	switch ingressName {
	case gloo:
		// Install and inject booksapp
		utils.TestBooksappApp()
		utils.TestBooksappInject()
	case contour:
		ginkgo.By("Installing and injecting sample application [kuard]")
		out, stderr, err := h.LinkerdRun("inject", kuardYAML)
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to inject sample application [kuard]: %s", stderr))

		_, err = h.Kubectl(out, "apply", "-f", "-")
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to install sample application [kuard]: %s", utils.Err(err)))

		// verify kuard installation
		err = h.CheckDeployment(kuardNs, kuard, 3)
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("deploy/%s in namespace/%s does not have expected replicas", kuard, kuardNs))
		err = utils.CheckProxyContainer(kuard, kuardNs)
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("could not find proxy container for pods under deploy/%s", kuard))
	default:
		utils.TestEmojivotoApp()
		utils.TestEmojivotoInject()
	}
}

func testSampleAppUninstall(ingressName string) {
	h, _ := utils.GetHelperAndConfig()

	switch ingressName {
	case gloo:
		utils.TestBooksappUninstall()

	case contour:
		_, err := h.Kubectl("", "delete", "-f", kuardYAML)
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("could not delete sample application [kuard]: %s", utils.Err(err)))
	default:
		utils.TestEmojivotoUninstall()
	}
}

func testControllerInstall(ingressName, controllerYAML, controllerDeployName, controllerNs string) {
	h, _ := utils.GetHelperAndConfig()
	ginkgo.By("Setting up ingress controller for Linkerd")
	installFailMessage := "failed to install ingress controller"

	switch ingressName {
	case gloo:
		err := utils.InstallGlooctlBinary()
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("could not install `glooctl` binary: %s", utils.Err(err)))

		_, err = utils.GlooctlRun("install", "gateway")
		gomega.Expect(err).Should(gomega.BeNil(), utils.Err(err))

	case contour:
		_, err := h.Kubectl("", "apply", "-f", controllerYAML) // YAML contains inject annotation
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("%s: %s", installFailMessage, utils.Err(err)))

		// verify contour installation
		err = h.CheckDeployment(controllerNs, controllerNs, 2)
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("deploy/%s in namespace/%s does not have expected replicas", controllerDeployName, controllerNs))
		err = utils.CheckProxyContainer(controllerDeployName, controllerNs)
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("could not find proxy container for pods under deploy/%s", controllerDeployName))
	default:
		_, err := h.Kubectl("", "apply", "-f", controllerYAML)
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("%s: %s", installFailMessage, utils.Err(err)))

		err = h.CheckPods(controllerNs, controllerDeployName, 1)
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to verify %s controller pods: %s", ingressName, utils.Err(err)))

		out, err := h.Kubectl("", "get", "-n", controllerNs, "deploy", controllerDeployName, "-o", "yaml")
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to get YAML manifest for deploy/%s: %s", controllerDeployName, utils.Err(err)))

		out, stderr, err := h.PipeToLinkerdRun(out, "inject", "-")
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to inject controller: %s", stderr))

		_, err = h.KubectlApply(out, controllerNs)
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to apply injected manifests: %s", utils.Err(err)))

		err = h.CheckPods(controllerNs, controllerDeployName, 1)
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to verify %s controller pods: %s", ingressName, utils.Err(err)))

		err = utils.CheckProxyContainer(controllerDeployName, controllerNs)
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("controller deployment does not contain proxy sidecar: %s", utils.Err(err)))
	}
}

func testControllerUninstall(ingressName, controllerYAML, controllerNs string) {
	h, _ := utils.GetHelperAndConfig()

	ginkgo.By(fmt.Sprintf("Uninstalling %s ingress controller", ingressName))
	failMessage := "could not delete ingress controller"

	switch ingressName {
	case gloo:
		_, err := utils.GlooctlRun("uninstall", "gateway")
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("%s: %s", failMessage, utils.Err(err)))

		_, err = h.Kubectl("", "delete", "ns", controllerNs)
		gomega.Expect(err).Should(gomega.BeNil(), utils.Err(err))

	case contour:
		_, err := h.Kubectl("", "delete", "--ignore-not-found", "-f", controllerYAML)
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("%s: %s", failMessage, utils.Err(err)))

	default:
		_, err := h.Kubectl("", "delete", "-f", controllerYAML)
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("%s: %s", failMessage, utils.Err(err)))
	}
}

func testIngress(ingressName, deploy, ns, resourceYAMLPath string) {
	h, _ := utils.GetHelperAndConfig()

	ginkgo.By("Applying ingress resource")
	_, err := h.Kubectl("", "apply", "-f", resourceYAMLPath)
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to create %s ingress resource: %s", ingressName, utils.Err(err)))

	ginkgo.By("Checking if emojivoto is reachable")
	ip, err := getExternalIP(deploy, ns)
	gomega.Expect(err).Should(gomega.BeNil(), utils.Err(err))

	err = h.RetryFor(3*time.Minute, func() error {
		return pingIP(fmt.Sprintf("http://%s", strings.Trim(ip, "'")))
	})

	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to reach emojivoto: %s", utils.Err(err)))

}

func testNginx() {
	var (
		nginxResourceYAML = "testdata/ingress/resources/nginx.yaml"
	)
	testIngress(nginx, nginxController, nginxNs, nginxResourceYAML)
}

func testTraefik() {
	var (
		traefikResourceYAML = "testdata/ingress/resources/traefik.yaml"
	)
	testIngress(traefik, traefikController, traefikNs, traefikResourceYAML)
}

func testAmbassador() {
	var (
		ambassadorResourceYAML = "testdata/ingress/resources/ambassador.yaml"
	)
	testIngress(ambassador, ambassadorController, ambassadorNs, ambassadorResourceYAML)
}

func testGloo() {
	h, _ := utils.GetHelperAndConfig()

	ginkgo.By("Enabling native integration with Linkerd")
	_, err := h.Kubectl("", "patch", "settings", "-n", glooNs, "default", "-p", "{\"spec\":{\"linkerd\":true}}", "--type", "merge")
	gomega.Expect(err).Should(gomega.BeNil(), utils.Err(err))

	ginkgo.By("Adding booksapp route to the virtual service")
	_, err = utils.GlooctlRun("add", "route", "--path-prefix", "/", "--dest-name", "booksapp-webapp-7000")
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to add booksapp route to virtual service: %s", utils.Err(err)))

	ginkgo.By("Checking if booksapp is reachable")

	err = h.RetryFor(time.Minute*5, func() error {
		ip, err := utils.GlooctlRun("proxy", "url")
		if err != nil {
			return fmt.Errorf("failed to fetch external IP: %s", err.Error())
		}

		err = pingIP(strings.Trim(ip, "'"))
		if err != nil {
			return fmt.Errorf("sample application booksapp is not reachable")
		}
		return nil
	})
}

func kubectlRunPortforward() (*exec.Cmd, error) {
	// kubectl port-forward svc/envoy -n projectcontour 3200:80
	cmd := exec.Command("kubectl", "port-forward", "svc/envoy", "-n", contourNs, "3200:80")
	if err := cmd.Start(); err != nil {
		return cmd, err
	}
	return cmd, nil
}

func testContour() {
	h, _ := utils.GetHelperAndConfig()
	var (
		contourResourceYAML = "testdata/ingress/resources/contour.yaml"
	)

	ginkgo.By("Install Contour resource to route traffic into sample application")
	_, err := h.Kubectl("", "apply", "-f", contourResourceYAML)
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to create ingress resource: %s", utils.Err(err)))

	ginkgo.By("Verifying if sample application [kuard] is reachable")
	// kubectl port-forward svc/envoy -n projectcontour 3200:80
	process, err := kubectlRunPortforward()
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to enable port-forward:%s", utils.Err(err)))

	err = h.RetryFor(time.Minute*5, func() error {
		err = pingIP("http://127.0.0.1.xip.io:3200")
		if err != nil {
			return fmt.Errorf("sample application booksapp is not reachable")
		}
		return nil
	})
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("could not reach sample application [kuard]: %s", utils.Err(err)))
	process.Process.Kill()

	_, err = h.Kubectl("", "delete", "-f", contourResourceYAML)
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("could not delete ingress resource: %s", utils.Err(err)))
}
