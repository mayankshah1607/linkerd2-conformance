package serviceprofiles

import (
	"fmt"
	"strings"
	"time"

	"github.com/linkerd/linkerd2-conformance/utils"
	cmd2 "github.com/linkerd/linkerd2/cli/cmd"
	sp "github.com/linkerd/linkerd2/controller/gen/apis/serviceprofile/v1alpha2"
	"github.com/linkerd/linkerd2/testutil"
	"github.com/onsi/gomega"
	"sigs.k8s.io/yaml"
)

const (
	booksappNs  = "booksapp"
	booksappURL = "https://run.linkerd.io/booksapp.yml"
)

var booksappDeploys = map[string]int{
	"authors": 1,
	"books":   1,
	"traffic": 1,
	"webapp":  3,
}

func testSampleApp() {
	h, _ := utils.GetHelperAndConfig()
	out, err := h.Kubectl("", "create", "ns", booksappNs)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to create namespace \"booksapp\": %s\n%s", out, err))

	out, err = h.Kubectl("",
		"-n", booksappNs,
		"apply", "-f", booksappURL)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to install booksapp: %s\n%s", out, err))

	for k, v := range booksappDeploys {
		err := h.RetryFor(5*time.Minute, func() error {
			err := h.CheckDeployment(booksappNs, k, v)
			if err != nil {
				return err
			}
			return nil
		})
		gomega.Expect(err).Should(gomega.BeNil(),
			fmt.Sprintf("CheckDeployment timed-out: %s", err))
	}

	deploys, err := h.Kubectl("",
		"-n", booksappNs,
		"get", "deploy",
		"-o", "yaml")
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`kubectl get` command failed: %s\n%s: ", deploys, err))

	injected, stderr, err := h.PipeToLinkerdRun(deploys, "inject", "-")
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`linkerd inject` command failed: %s\n%s: ", injected, stderr))

	out, err = h.Kubectl(injected, "-n", booksappNs, "apply", "-f", "-")
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`kubectl apply` command failed: %s\n%s: ", out, err))

	for k, v := range booksappDeploys {
		err = h.RetryFor(5*time.Minute, func() error {
			err := h.CheckPods(booksappNs, k, v)
			if err != nil {
				return err
			}

			err = h.CheckDeployment(booksappNs, k, v)
			if err != nil {
				return err
			}
			return nil
		})
		gomega.Expect(err).Should(gomega.BeNil(),
			fmt.Sprintf("CheckDeployment timed-out: %s", err))

		err := utils.CheckProxyContainer(k, booksappNs)
		gomega.Expect(err).Should(gomega.BeNil(), err)
	}
}

func testInstallSP() {
	h, _ := utils.GetHelperAndConfig()

	const (
		authorsSwagger = "testdata/serviceprofiles/authors.swagger"
		booksSwagger   = "testdata/serviceprofiles/books.swagger"
		webappSwagger  = "testdata/serviceprofiles/webapp.swagger"
	)

	apiSpecs := map[string]string{
		"authors": authorsSwagger,
		"books":   booksSwagger,
		"webapp":  webappSwagger,
	}
	for deploy, spec := range apiSpecs {
		out, err := testutil.ReadFile(spec)
		gomega.Expect(err).Should(gomega.BeNil(),
			fmt.Sprintf("failed to read file %s: %s", spec, err))

		out, stderr, err := h.PipeToLinkerdRun(out, "-n", booksappNs,
			"profile", "--open-api",
			"-",
			deploy)
		gomega.Expect(err).Should(gomega.BeNil(),
			fmt.Sprintf("`linkerd profile` command failed: %s\n%s", out, stderr))

		out, err = h.KubectlApply(out, booksappNs)
		gomega.Expect(err).Should(gomega.BeNil(),
			fmt.Sprintf("`kubectl apply` command failed: %s\n%s", out, err))
	}
}

func testRoutes() {
	testCases := []struct {
		source      string
		destination string
		routes      []string
	}{
		{
			source:      "deployment/webapp",
			destination: "service/books",
			routes: []string{
				"DELETE /books/{id}.json",
				"GET /books.json",
				"GET /books/{id}.json",
				"POST /books.json",
				"PUT /books/{id}.json",
				"[DEFAULT]",
			},
		},
		{
			source:      "deployment/webapp",
			destination: "service/authors",
			routes: []string{
				"DELETE /authors/{id}.json",
				"GET /authors.json",
				"GET /authors/{id}.json",
				"HEAD /authors/{id}.json",
				"POST /authors.json",
				"[DEFAULT]",
			},
		},
		{
			source:      "deployment/authors",
			destination: "service/books",
			routes: []string{
				"DELETE /books/{id}.json",
				"GET /books.json",
				"GET /books/{id}.json",
				"POST /books.json",
				"PUT /books/{id}.json",
				"[DEFAULT]",
			},
		},
		{
			source:      "deployment/books",
			destination: "service/authors",
			routes: []string{
				"DELETE /authors/{id}.json",
				"GET /authors.json",
				"GET /authors/{id}.json",
				"HEAD /authors/{id}.json",
				"POST /authors.json",
				"[DEFAULT]",
			},
		},
	}

	for _, tc := range testCases {
		routes, err := getRoutes(tc.source, tc.destination, booksappNs, []string{})
		gomega.Expect(err).Should(gomega.BeNil(),
			fmt.Sprintf("`linkerd routes` command failed: %s", err))

		err = assertExpectedRoutes(tc.routes, routes)
		gomega.Expect(err).Should(gomega.BeNil(), fmt.Sprintf("failed to match routes: %s", err))
	}
}

func testRetries() {
	const (
		sourceDeploy      = "deployment/books"
		destinationDeploy = "service/authors"
		targetRoute       = "HEAD /authors/{id}.json"
	)
	h, _ := utils.GetHelperAndConfig()

	err := assertRouteStat(sourceDeploy, destinationDeploy, booksappNs, targetRoute, func(stat *cmd2.JSONRouteStats) error {
		if !(*stat.ActualSuccess < 55) {
			return fmt.Errorf("expected effective sucess to less than 55%%, found %0.2f",
				*stat.ActualSuccess)
		}
		return nil
	})

	gomega.Expect(err).Should(gomega.BeNil(), utils.Err(err))

	profile := &sp.ServiceProfile{}

	out, err := h.Kubectl("",
		"get",
		"-n", booksappNs,
		"sp/authors.booksapp.svc.cluster.local",
		"-o", "yaml")

	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`kubectl get sp` command failed: %s", err))

	err = yaml.Unmarshal([]byte(out), profile)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to unmarshal ServiceProfile object: %s", err))

	// make target route retryable
	for _, route := range profile.Spec.Routes {
		if route.Name == targetRoute {
			route.IsRetryable = true
		}
	}

	bytes, err := yaml.Marshal(profile)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to marshal ServiceProfile object: %s", err))

	out, err = h.KubectlApply(string(bytes), booksappNs)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`kubectl apply` command failed: %s\n%s", out, err))

	err = assertRouteStat(sourceDeploy, destinationDeploy, booksappNs, targetRoute, func(stat *cmd2.JSONRouteStats) error {
		if *stat.EffectiveSuccess < 0.95 {
			return fmt.Errorf("expected effective sucess to be at least 95%%, found %0.2f",
				*stat.EffectiveSuccess)
		}
		return nil
	})
	gomega.Expect(err).Should(gomega.BeNil(), utils.Err(err))
}

func assertRouteStat(source, destination, namespace, targetRoute string, assertFn func(stat *cmd2.JSONRouteStats) error) error {
	h, _ := utils.GetHelperAndConfig()

	err := h.RetryFor(time.Minute*5, func() error {
		routes, err := getRoutes(source, destination, namespace, []string{})
		if err != nil {
			return err
		}

		for _, r := range routes {
			if r.Route == targetRoute {
				return assertFn(r)
			}
		}
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

func assertExpectedRoutes(expected []string, actual []*cmd2.JSONRouteStats) error {
	if len(expected) != len(actual) {
		return fmt.Errorf("mismatch routes count. Expected %d, Actual %d", len(expected), len(actual))
	}

	for _, expectedRoute := range expected {
		containsRoute := false
		for _, actualRoute := range actual {
			if actualRoute.Route == expectedRoute {
				containsRoute = true
				break
			}
		}

		if !containsRoute {
			sb := strings.Builder{}
			for _, route := range actual {
				sb.WriteString(fmt.Sprintf("%s ", route.Route))
			}
			return fmt.Errorf("expected route %s not found in %+v", expectedRoute, sb.String())
		}
	}
	return nil
}

func getRoutes(source, destination, namespace string, additionalArgs []string) ([]*cmd2.JSONRouteStats, error) {

	cmd := []string{"routes", "--namespace", namespace, source, "--to", destination}
	h, _ := utils.GetHelperAndConfig()

	if len(additionalArgs) > 0 {
		cmd = append(cmd, additionalArgs...)

	}

	cmd = append(cmd, "--output", "json")
	var out, stderr string
	err := h.RetryFor(2*time.Minute, func() error {
		var err error
		out, stderr, err = h.LinkerdRun(cmd...)
		return err

	})
	if err != nil {
		return nil, err

	}

	var list map[string][]*cmd2.JSONRouteStats
	err = yaml.Unmarshal([]byte(out), &list)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("Error: %s stderr: %s", err, stderr))

	}

	if deployment, ok := list[source]; ok {
		return deployment, nil

	}
	return nil, fmt.Errorf("could not retrieve route info for %s", source)

}

func testClean() {
	h, _ := utils.GetHelperAndConfig()

	out, err := h.Kubectl("", "get",
		"all", "-n", booksappNs,
		"-o", "yaml")
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`kubectl get all` command failed: %s\n%s", out, err))

	out, err = h.Kubectl(out, "delete", "-f", "-")
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`kubectl delete` command failed: %s\n%s", out, err))

	out, err = h.Kubectl("", "delete", "ns", booksappNs)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`kubectl delete namespace` command failed: %s\n%s", out, err))
}
