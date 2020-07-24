package utils

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/linkerd/linkerd2/testutil"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

var (
	linkerdSvcs = []string{
		"linkerd-controller-api",
		"linkerd-dst",
		"linkerd-grafana",
		"linkerd-identity",
		"linkerd-prometheus",
		"linkerd-web",
		"linkerd-tap",
	}
)

// CheckOutput is used for unmarshalling the
// output from `linkerd check -o json`
type CheckOutput struct {
	Success    bool `json:"success"`
	Categories []struct {
		CategoryName string `json:"categoryName"`
		Checks       []struct {
			Result string `json:"result"`
			Error  string `json:"error"`
		}
	}
}

func getFailedChecks(r *CheckOutput) string {
	err := "The following errors were detected:\n"

	for _, c := range r.Categories {
		for _, check := range c.Checks {
			if check.Result == "error" {
				err = fmt.Sprintf("%s\n%s", err, check.Error)
			}
		}
	}

	return err
}

// RunCheck rus `linkerd check`
func RunCheck(h *testutil.TestHelper, pre bool) {

	var checkResult *CheckOutput

	cmd := []string{
		"check",
		"-o",
		"json",
	}

	if pre {
		cmd = append(cmd, "--pre")
		ginkgo.By("Running pre-installation checks")
	} else {
		ginkgo.By("Running post-installation checks")
	}

	out, _, _ := h.LinkerdRun(cmd...)

	ginkgo.By("Validating `check` output")
	err := json.Unmarshal([]byte(out), &checkResult)
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to unmarshal check results JSON: %s", Err(err)))
	gomega.Expect(checkResult.Success).Should(gomega.BeTrue(), fmt.Sprintf("`linkerd check failed: %s`\n Check errors: %s", Err(err), getFailedChecks(checkResult)))
}

// InstallLinkerdControlPlane runs the control plane install tests
func InstallLinkerdControlPlane(h *testutil.TestHelper, c *ConformanceTestOptions) {
	withHA := c.HA()

	ginkgo.By(fmt.Sprintf("Installing linkerd control plane with HA: %v", withHA))
	RunCheck(h, true) // run pre checks

	if err := h.CheckIfNamespaceExists(h.GetLinkerdNamespace()); err == nil {
		ginkgo.Skip(fmt.Sprintf("linkerd control plane already exists in namespace %s", h.GetLinkerdNamespace()))
	}

	// TODO: Uncomment while writing Helm tests
	// ginkgo.By("verifying if Helm release is empty")
	// gomega.Expect(h.GetHelmReleaseName()).To(gomega.Equal(""))

	cmd := "install"
	args := []string{}

	// parse install flags from config
	for _, flag := range c.GetInstallFlags() {
		args = append(args, flag)
	}

	if len(c.GetAddons()) > 0 {

		addOnFile := "../../addons.yaml"
		if !fileExists(addOnFile) {
			out, err := c.GetAddOnsYAML()
			gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to produce add-on config file: %s", Err(err)))

			err = createFileWithContent(out, addOnFile)
			gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to write add-ons to YAML: %s", Err(err)))
		}

		ginkgo.By(fmt.Sprintf("Using add-ons file %s", addOnFile))
		args = append(args, "--addon-config")
		args = append(args, addOnFile)
	}

	if withHA {
		args = append(args, "--ha")
	}

	if h.GetClusterDomain() != "cluster.local" {
		args = append(args, "--cluster-domain", h.GetClusterDomain())
	}

	exec := append([]string{cmd}, args...)

	ginkgo.By("Running `linkerd install`")
	out, stderr, err := h.LinkerdRun(exec...)
	gomega.Expect(err).Should(gomega.BeNil(), stderr)

	ginkgo.By("Applying control plane manifests")
	out, err = h.KubectlApply(out, "")
	gomega.Expect(err).Should(gomega.BeNil(), Err(err))

	TestControlPlanePostInstall(h)
	RunCheck(h, false) // run post checks
}

// UninstallLinkerdControlPlane runs the test for
// control plane uninstall
func UninstallLinkerdControlPlane(h *testutil.TestHelper) {
	ginkgo.By("Uninstalling linkerd control plane")
	cmd := "install"
	args := []string{
		"--ignore-cluster",
	}

	exec := append([]string{cmd}, args...)

	ginkgo.By("Gathering control plane manifests")
	out, stderr, err := h.LinkerdRun(exec...)
	gomega.Expect(err).Should(gomega.BeNil(), stderr)

	args = []string{"delete", "-f", "-"}

	ginkgo.By("Deleting resources from the cluster")
	out, err = h.Kubectl(out, args...)
	gomega.Expect(err).Should(gomega.BeNil(), Err(err))

	RunCheck(h, true) // run pre checks
}

func testResourcesPostInstall(namespace string, services []string, deploys map[string]testutil.DeploySpec, h *testutil.TestHelper) {
	ginkgo.By(fmt.Sprintf("Checking resources in namespace %s", namespace))
	err := h.CheckIfNamespaceExists(namespace)
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("could not find namespace %s", namespace))

	for _, svc := range services {
		err = h.CheckService(namespace, svc)
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("error validating service %s: %s", svc, Err(err)))
	}

	for deploy, spec := range deploys {
		err = h.CheckPods(namespace, deploy, spec.Replicas)
		if err != nil {
			if _, ok := err.(*testutil.RestartCountError); !ok { // if error is not due to restart count
				ginkgo.Fail(fmt.Sprintf("CheckPods timed-out: %s", Err(err)))
			}
		}

		err := h.CheckDeployment(namespace, deploy, spec.Replicas)
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("CheckDeployment timed-out for deploy/%s: %s", deploy, Err(err)))

	}
}

// TestControlPlanePostInstall tests the control plane resources post installation
func TestControlPlanePostInstall(h *testutil.TestHelper) {
	testResourcesPostInstall(h.GetLinkerdNamespace(), linkerdSvcs, testutil.LinkerdDeployReplicas, h)
}

// RunBeforeAndAfterEachSetup runs the control plane installation
// and uninstallation tests when a new control plane is required by each test
func RunBeforeAndAfterEachSetup() {
	h, c := GetHelperAndConfig()
	if !c.SingleControlPlane() {
		_ = ginkgo.BeforeEach(func() {
			InstallLinkerdControlPlane(h, c)
		})

		_ = ginkgo.AfterEach(func() {
			UninstallLinkerdControlPlane(h)
		})
	}
}

func checkSampleAppState(deploys map[string]int, appNs string) error {
	h, _ := GetHelperAndConfig()
	for deploy, count := range deploys {
		if err := h.CheckPods(appNs, deploy, count); err != nil {
			if _, ok := err.(*testutil.RestartCountError); !ok { // err is not due to restart
				return fmt.Errorf("failed to validate emojivoto pods: %s", err.Error())
			}
		}

		err := h.CheckDeployment(appNs, deploy, 1)
		if err != nil {
			fmt.Sprintf("failed to validate deploy/%s: %s", deploy, err.Error())
		}
	}
	return nil
}

func installApp(manifestPath, appNs string) error {
	h, _ := GetHelperAndConfig()

	resources, err := testutil.ReadFile(manifestPath)
	if err != nil {
		return err
	}

	_, err = h.KubectlApply(resources, appNs)
	if err != nil {
		return err
	}
	return nil
}

// TestEmojivotoApp installs and checks if emojivoto app is installed
// called of the function must have `testdata/emojivoto.yml`
func TestEmojivotoApp() {
	h, _ := GetHelperAndConfig()

	ginkgo.By("Installing emojivoto [sample application]")
	err := installApp("testdata/emojivoto.yml", emojivotoNs)
	gomega.Expect(err).Should(gomega.BeNil(), Err(err))

	err = checkSampleAppState(emojivotoDeploys, emojivotoNs)
	gomega.Expect(err).Should(gomega.BeNil(), Err(err))

	err = testutil.ExerciseTestAppEndpoint("/api/list", emojivotoNs, h)
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to exercise emojivoto endpoint: %s", Err(err)))
}

func TestBooksappApp() {
	ginkgo.By("Installing booksapp [sample application]")
	err := installApp("testdata/booksapp.yaml", booksappNs)
	gomega.Expect(err).Should(gomega.BeNil(), Err(err))

	err = checkSampleAppState(booksappDeploys, booksappNs)
	gomega.Expect(err).Should(gomega.BeNil(), Err(err))
}

//TestEmojivotoInject installs and checks if emojivoto app is installed
func TestEmojivotoInject() {
	sampleAppInject(emojivotoNs, emojivotoDeploys)
}

func TestBooksappInject() {
	sampleAppInject(booksappNs, booksappDeploys)
}

// TestEmojivotoUninstall tests if emojivoto can be successfull uninstalled
func TestEmojivotoUninstall() {
	sampleAppUninstall(emojivotoNs)
}

// TestBooksappUninstall tests if booksapp can be successfully uninjected
func TestBooksappUninstall() {
	sampleAppUninstall(booksappNs)
}

func sampleAppInject(appNs string, deploys map[string]int) {
	ginkgo.By(fmt.Sprintf("Injecting sample app in namespace/%s", appNs))
	h, _ := GetHelperAndConfig()

	out, err := h.Kubectl("", "get", "deploy", "-n", appNs, "-o", "yaml")
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to get manifests: %s", Err(err)))

	out, stderr, err := h.PipeToLinkerdRun(out, "inject", "-")
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to inject: %s", stderr))

	out, err = h.KubectlApply(out, appNs)
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to apply injected resources: %s", Err(err)))

	err = checkSampleAppState(deploys, appNs)
	gomega.Expect(err).Should(gomega.BeNil(), Err(err))

	for deploy := range deploys {
		err := CheckProxyContainer(deploy, appNs)
		gomega.Expect(err).Should(gomega.BeNil(), Err(err))
	}
}

func sampleAppUninstall(appNs string) {
	ginkgo.By(fmt.Sprintf("Uninstalling sample application in namespace %s", appNs))
	h, _ := GetHelperAndConfig()

	_, err := h.Kubectl("", "delete", "ns", appNs)
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("could not uninstall sample app in namespace %s: %s", appNs, Err(err)))
}

// CheckProxyContainer gets the pods from a deployment, and checks if the proxy container is present
func CheckProxyContainer(deployName, namespace string) error {
	h, _ := GetHelperAndConfig()
	return h.RetryFor(time.Minute*3, func() error {
		pods, err := h.GetPodsForDeployment(namespace, deployName)
		if err != nil || len(pods) == 0 {
			return fmt.Errorf("could not get pod(s) for deployment %s: %s", deployName, err.Error())
		}
		containers := pods[0].Spec.Containers
		if len(containers) == 0 {
			return fmt.Errorf("could not find container(s) for deployment %s", deployName)
		}
		proxyContainer := testutil.GetProxyContainer(containers)
		if proxyContainer == nil {
			return fmt.Errorf("could not find proxy container for deployment %s", deployName)
		}
		return nil
	})
}

// ShouldTestSkip is called within a Describe block to determine if a test must be skipped
func ShouldTestSkip(skip bool, message string) bool {
	return ginkgo.BeforeEach(func() {
		if skip {
			ginkgo.Skip(message)
		}
	})
}
