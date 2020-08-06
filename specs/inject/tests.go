package inject

import (
	"fmt"
	"time"

	"github.com/linkerd/linkerd2-conformance/utils"
	"github.com/linkerd/linkerd2/pkg/k8s"
	"github.com/linkerd/linkerd2/testutil"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	proxyInjectTestNs           string
	nsAnnotationsOverrideTestNs string
)

func testInjectManual(withParams bool) {
	var golden string

	h, _ := utils.GetHelperAndConfig()

	injectYAMLPath := "testdata/inject/inject_test.yaml"
	cmd := []string{"inject",
		"--manual",
		"--linkerd-namespace=fake-ns",
		"--disable-identity",
		"--ignore-cluster",
		"--proxy-version=proxy-version",
		"--proxy-image=proxy-image",
		"--init-image=init-image",
	}

	if withParams {
		params := []string{
			"--disable-tap",
			"--image-pull-policy=Never",
			"--control-port=123",
			"--skip-inbound-ports=234,345",
			"--skip-outbound-ports=456,567",
			"--inbound-port=678",
			"--admin-port=789",
			"--outbound-port=890",
			"--proxy-cpu-request=10m",
			"--proxy-memory-request=10Mi",
			"--proxy-cpu-limit=20m",
			"--proxy-memory-limit=20Mi",
			"--proxy-uid=1337",
			"--proxy-log-level=warn",
			"--enable-external-profiles",
		}
		cmd = append(cmd, params...)

		golden = "inject/inject_params.golden"
	} else {
		golden = "inject/inject_default.golden"
	}
	cmd = append(cmd, injectYAMLPath)

	out, stderr, err := h.LinkerdRun(cmd...)
	utils.ExpectNil(err,
		"failed to run `linkerd inject`: %s\n%s",
		out, stderr)

	err = testutil.ValidateInject(out, golden, h)
	utils.ExpectNil(err, "failed to validate inject: %s", utils.Err(err))
}

func testProxyInjection() {
	h, _ := utils.GetHelperAndConfig()

	podYAML, err := testutil.ReadFile("testdata/inject/pod.yaml")
	utils.ExpectNil(err, "failed to read [pod.yaml]: %s", utils.Err(err))

	injectNs := "inject-pod-test"
	podName := "inject-pod-test-terminus"
	nsAnnotations := map[string]string{
		k8s.ProxyInjectAnnotation: k8s.ProxyInjectEnabled,
	}

	proxyInjectTestNs = h.GetTestNamespace(injectNs)
	err = h.CreateDataPlaneNamespaceIfNotExists(proxyInjectTestNs, nsAnnotations)
	utils.ExpectNil(err,
		"failed to create namespace %s: %s",
		proxyInjectTestNs, utils.Err(err))

	o, err := h.Kubectl(podYAML, "-n", proxyInjectTestNs, "create", "-f", "-")
	utils.ExpectNil(err,
		"failed to create pod/%s in namespace %s: %s\n%s",
		podName, proxyInjectTestNs, o, utils.Err(err))

	o, err = h.Kubectl("",
		"-n",
		proxyInjectTestNs,
		"wait",
		"--for=condition=initialized",
		"--timeout=120s",
		"pod/"+podName)
	utils.ExpectNil(err,
		"failed to wait for pod/%s to be initialized in namespace %s: %s\n%s",
		podName, proxyInjectTestNs, o, utils.Err(err))

	err = h.RetryFor(time.Minute*3, func() error {
		pods, err := h.GetPods(proxyInjectTestNs, map[string]string{"app": podName})
		if err != nil {
			return fmt.Errorf("failed to fetch pod/%s: %s", podName, err.Error())
		}

		containers := pods[0].Spec.Containers

		proxyContainers := testutil.GetProxyContainer(containers)
		if proxyContainers == nil {
			return fmt.Errorf("proxy container is not injected")
		}
		return nil
	})
	utils.ExpectNil(err, utils.Err(err))
}

func testInjectAutoNsOverrideAnnotations() {

	h, _ := utils.GetHelperAndConfig()

	injectYAML, err := testutil.ReadFile("testdata/inject/inject_test.yaml")
	utils.ExpectNil(err, "failed to read [inject_test]: %s", utils.Err(err))

	injectNs := "inj-ns-override-test"
	deployName := "inject-test-terminus"
	nsProxyMemReq := "50Mi"
	nsProxyCPUReq := "200m"

	nsAnnotations := map[string]string{
		k8s.ProxyInjectAnnotation:        k8s.ProxyInjectEnabled,
		k8s.ProxyCPURequestAnnotation:    nsProxyCPUReq,
		k8s.ProxyMemoryRequestAnnotation: nsProxyMemReq,
	}

	nsAnnotationsOverrideTestNs = h.GetTestNamespace(injectNs)
	err = h.CreateDataPlaneNamespaceIfNotExists(nsAnnotationsOverrideTestNs, nsAnnotations)
	utils.ExpectNil(err,
		"failed to create namespace %s: %s",
		nsAnnotations, utils.Err(err))

	podProxyCPUReq := "600m"
	podAnnotations := map[string]string{
		k8s.ProxyCPURequestAnnotation: podProxyCPUReq,
	}

	patchedYAML, err := testutil.PatchDeploy(injectYAML, deployName, podAnnotations)
	utils.ExpectNil(err,
		"failed to patch inject test YAML in namespace %s for deploy/%s: %s",
		nsAnnotationsOverrideTestNs, deployName, utils.Err(err))

	o, err := h.Kubectl(patchedYAML, "-n", nsAnnotationsOverrideTestNs, "create", "-f", "-")
	utils.ExpectNil(err,
		"failed to create deploy/%s in namespace %s for  %s: %s",
		deployName, nsAnnotationsOverrideTestNs, o, utils.Err(err))

	o, err = h.Kubectl("",
		"--namespace",
		nsAnnotationsOverrideTestNs,
		"wait",
		"--for=condition=available",
		"--timeout=120s",
		"deploy/"+deployName)
	utils.ExpectNil(err,
		"failed to wait for deploy/%s in namespace %s for  %s: %s",
		deployName, nsAnnotationsOverrideTestNs, o, utils.Err(err))

	pods, err := h.GetPodsForDeployment(nsAnnotationsOverrideTestNs, deployName)
	utils.ExpectNil(err,
		"failed to get pods for namespace %s: %s",
		nsAnnotationsOverrideTestNs, utils.Err(err))

	containers := pods[0].Spec.Containers
	proxyContainer := testutil.GetProxyContainer(containers)

	utils.ExpectEqual(proxyContainer.Resources.Requests["memory"],
		resource.MustParse(nsProxyMemReq),
		"proxy memory resource request failed to match with namespace level override")

	utils.ExpectEqual(proxyContainer.Resources.Requests["cpu"],
		resource.MustParse(podProxyCPUReq),
		"proxy cpu resource request failed to match with namespace level override")
}

func testClean() {
	h, _ := utils.GetHelperAndConfig()

	namespaces := []string{
		proxyInjectTestNs,
		nsAnnotationsOverrideTestNs,
	}

	for _, ns := range namespaces {
		out, err := h.Kubectl("", "-n", ns, "get", "all", "-o", "yaml")
		utils.ExpectNil(err,
			"`kubectl get` command failed: %s\n%s",
			out,
			utils.Err(err))

		out, err = h.Kubectl(out, "delete", "-f", "-")
		utils.ExpectNil(err,
			"`kubectl delete` command failed: %s\n%s",
			out,
			utils.Err(err))

		out, err = h.Kubectl("", "delete", "ns", ns)
		utils.ExpectNil(err,
			"`kubectl delete ns` command failed: %s\n%s",
			out,
			utils.Err(err))
	}
}
