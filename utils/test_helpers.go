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
	}

	out, stderr, err := h.LinkerdRun(cmd...)
	ExpectNil(err,
		"`linkerd check` command failed: %s\n%s",
		out, stderr)

	err = json.Unmarshal([]byte(out), &checkResult)
	ExpectNil(err, "failed to unmarshal check results JSON: %s", Err(err))
	ExpectTrue(checkResult.Success,
		"`linkerd check failed: %s`\n Check errors: %s",
		Err(err), getFailedChecks(checkResult))
}

// InstallLinkerdControlPlane runs the control plane install tests
func InstallLinkerdControlPlane(h *testutil.TestHelper, c *ConformanceTestOptions) {
	withHA := c.HA()

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
	args = append(args, c.GetInstallFlags()...)

	if len(c.GetAddons()) > 0 {

		addOnFile := "../../addons.yaml"
		if !fileExists(addOnFile) {
			out, err := c.GetAddOnsYAML()
			ExpectNil(err,
				"failed to produce add-on config file: %s\n%s",
				out, Err(err))

			err = createFileWithContent(out, addOnFile)
			ExpectNil(err, "failed to write add-ons to YAML: %s", Err(err))
		}

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

	out, stderr, err := h.LinkerdRun(exec...)
	ExpectNil(err,
		"`linkerd install` command failed: %s\n%s", out, stderr)

	out, err = h.KubectlApply(out, "")
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to apply manifests: %s\n%s", Err(err), out))

	TestControlPlanePostInstall(h)
	RunCheck(h, false) // run post checks
}

// UninstallLinkerdControlPlane runs the test for
// control plane uninstall
func UninstallLinkerdControlPlane(h *testutil.TestHelper) {
	cmd := "install"
	args := []string{
		"--ignore-cluster",
	}

	exec := append([]string{cmd}, args...)

	out, stderr, err := h.LinkerdRun(exec...)
	ExpectNil(err, "`linkerd install` command failed: %s\n%s", out, stderr)
	args = []string{"delete", "--ignore-not-found", "-f", "-"}

	out, err = h.Kubectl(out, args...)
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to delete resources: %s\n%s", Err(err), out))

	RunCheck(h, true) // run pre checks
}

func testResourcesPostInstall(namespace string, services []string, deploys map[string]testutil.DeploySpec, h *testutil.TestHelper) {
	err := h.CheckIfNamespaceExists(namespace)
	ExpectNil(err, "could not find namespace %s", namespace)

	for _, svc := range services {
		err = h.CheckService(namespace, svc)
		ExpectNil(err, "error validating service %s: %s", svc, Err(err))
	}

	for deploy, spec := range deploys {
		err = h.CheckPods(namespace, deploy, spec.Replicas)
		if err != nil {
			if _, ok := err.(*testutil.RestartCountError); !ok { // if error is not due to restart count
				ginkgo.Fail(fmt.Sprintf("CheckPods timed-out: %s", Err(err)))
			}
		}

		err := h.CheckDeployment(namespace, deploy, spec.Replicas)
		ExpectNil(err, "CheckDeploement timed-out for deploy/%s: %s", deploy, Err(err))

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

var (
	emojivotoNs      = "emojivoto"
	emojivotoDeploys = []string{"emoji", "voting", "web"}
)

func checkSampleAppState() {
	h, _ := GetHelperAndConfig()
	for _, deploy := range emojivotoDeploys {
		if err := h.CheckPods(emojivotoNs, deploy, 1); err != nil {
			if _, ok := err.(*testutil.RestartCountError); !ok { // err is not due to restart
				ginkgo.Fail(fmt.Sprintf("failed to validate emojivoto pods: %s", err.Error()))
			}
		}

		err := h.CheckDeployment(emojivotoNs, deploy, 1)
		ExpectNil(err, "failed to validate deploy/%s: %s", deploy, Err(err))
	}

	err := testutil.ExerciseTestAppEndpoint("/api/list", emojivotoNs, h)
	ExpectNil(err, "failed to exercise emojivoto endpoint: %s", Err(err))
}

// TestEmojivotoApp installs and checks if emojivoto app is installed
// called of the function must have `testdata/emojivoto.yml`
func TestEmojivotoApp() {
	h, _ := GetHelperAndConfig()
	resources, err := testutil.ReadFile("testdata/emojivoto.yml")
	ExpectNil(err, "failed to read [emojivoto.yml]: %s", Err(err))

	out, err := h.KubectlApply(resources, emojivotoNs)
	ExpectNil(err, "`kubectl apply` commadn failed: %s\n%s", out, Err(err))
	checkSampleAppState()
}

//TestEmojivotoInject installs and checks if emojivoto app is installed
// called of the function must have `testdata/emojivoto.yml`
func TestEmojivotoInject() {
	h, _ := GetHelperAndConfig()

	out, err := h.Kubectl("", "get", "deploy", "-n", emojivotoNs, "-o", "yaml")
	ExpectNil(err, "`kubectl get` command failed: %s\n%s", out, Err(err))

	out, stderr, err := h.PipeToLinkerdRun(out, "inject", "-")
	ExpectNil(err, "`linkerd inject` command failed: %s\n%s", out, stderr)

	out, err = h.KubectlApply(out, emojivotoNs)
	gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to apply injected resources: %s\n%s", Err(err), out))
	checkSampleAppState()

	for _, deploy := range emojivotoDeploys {
		err := CheckProxyContainer(deploy, emojivotoNs)
		gomega.Expect(err).Should(gomega.BeNil(), Err(err))
	}
}

// TestEmojivotoUninstall tests if emojivoto can be successfull uninstalled
func TestEmojivotoUninstall() {
	h, _ := GetHelperAndConfig()

	out, err := h.Kubectl("", "delete", "ns", emojivotoNs)
	ExpectNil(err, "`kubectl delete ns` command failed %s\n%s", out, Err(err))
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
