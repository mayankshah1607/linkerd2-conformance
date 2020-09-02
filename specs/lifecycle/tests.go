package lifecycle

import (
	"fmt"

	"github.com/linkerd/linkerd2-conformance/utils"
	"github.com/onsi/gomega"
)

func testUpgradeCLI() {
	h, c := utils.GetHelperAndConfig()

	err := utils.InstallLinkerdBinary(c.GetLinkerdPath(), h.GetVersion(), true, false)
	gomega.Expect(err).Should(gomega.BeNil(), utils.Err(err))

	cmd := []string{
		"version",
		"--short",
		"--client",
	}

	out, stderr, err := h.LinkerdRun(cmd...)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("could not run `linkerd version command`: %s", stderr))

	gomega.Expect(out).Should(gomega.ContainSubstring(h.GetVersion()),
		"failed to upgrade CLI")
}

func testUpgrade() {
	h, _ := utils.GetHelperAndConfig()

	cmd := "upgrade"

	out, stderr, err := h.LinkerdRun(cmd)
	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("`linkerd upgrade` command failed: %s", stderr))

	_, err = h.Kubectl(out, "apply", "--prune", "-l", "linkerd.io/control-plane-ns="+h.GetLinkerdNamespace(), "-f", "-")

	gomega.Expect(err).Should(gomega.BeNil(),
		fmt.Sprintf("failed to apply manifests: %s", utils.Err(err)))

	utils.TestControlPlanePostInstall(h)
	utils.RunCheck(h, false)
}
