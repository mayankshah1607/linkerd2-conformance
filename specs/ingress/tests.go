package ingress

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/linkerd/linkerd2-conformance/utils"
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

func testNginx() {
	h, _ := utils.GetHelperAndConfig()
	out, err := h.Kubectl("", "apply", "-f", "testdata/ingress/controllers/nginx.yaml")
	utils.ExpectNil(err,
		"`kubectl apply` command failed: %s\n%s", out, utils.Err(err))

	err = h.CheckPods(utils.NginxNs, utils.NginxController, 1)
	utils.ExpectNil(err,
		"failed to verify controller pods: %s", utils.Err(err))

	out, err = h.Kubectl("", "get", "-n", utils.NginxNs, "deploy", utils.NginxController, "-o", "yaml")
	utils.ExpectNil(err,
		"`kubectl get` command failed: %s\n%s", out, utils.Err(err))

	out, stderr, err := h.PipeToLinkerdRun(out, "inject", "-")
	utils.ExpectNil(err, "`linkerd inject` command failed: %s\n%s", out, stderr)

	out, err = h.KubectlApply(out, utils.NginxNs)
	utils.ExpectNil(err, "`kubectl apply` command failed: %s\n%s", out, utils.Err(err))

	err = h.CheckPods(utils.NginxNs, utils.NginxController, 1)
	utils.ExpectNil(err, "failed to verify controller pods: %s", utils.Err(err))

	// Wait upto 3mins for proxy container to show up
	err = utils.CheckProxyContainer(utils.NginxController, utils.NginxNs)
	utils.ExpectNil(err, utils.Err(err))

	out, err = h.Kubectl("", "apply", "-f", "testdata/ingress/resources/nginx.yaml")
	utils.ExpectNil(err,
		"`kubectl apply` command failed: %s\n%s",
		out, utils.Err(err))

	ip, err := getExternalIP(utils.NginxController, utils.NginxNs)
	utils.ExpectNil(err, utils.Err(err))

	err = h.RetryFor(3*time.Minute, func() error {
		return pingEmojivoto(strings.Trim(ip, "'"))
	})

	utils.ExpectNil(err, "failed to reach emojivoto: %s", utils.Err(err))

	out, err = h.Kubectl("", "delete", "ns", utils.NginxNs)
	utils.ExpectNil(err,
		"`kubectl delete` command failed: %s\n%s", out, utils.Err(err))
}
