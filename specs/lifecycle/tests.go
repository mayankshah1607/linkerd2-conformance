package lifecycle

import (
	"github.com/linkerd/linkerd2-conformance/utils"
)

func testUpgradeCLI() {
	h, c := utils.GetHelperAndConfig()

	err := utils.InstallLinkerdBinary(c.GetLinkerdPath(), h.GetVersion(), true, false)
	utils.ExpectNil(err,
		"failed to install Linkerd binary: %s", utils.Err(err))

	cmd := []string{
		"version",
		"--short",
		"--client",
	}

	out, stderr, err := h.LinkerdRun(cmd...)
	utils.ExpectNil(err,
		"failed to run `linkerd version` command: %s\n%s",
		out, stderr)

	utils.ExpectSubstring(out, h.GetVersion(), "failed to upgrade CLI")
}

func testUpgrade() {
	h, _ := utils.GetHelperAndConfig()

	cmd := "upgrade"

	out, stderr, err := h.LinkerdRun(cmd)
	utils.ExpectNil(err,
		"`linkerd upgrade` command failed: %s\n%s",
		out, stderr)

	out, err = h.Kubectl(out,
		"apply",
		"--prune",
		"-l",
		"linkerd.io/control-plane-ns="+h.GetLinkerdNamespace(),
		"-f",
		"-")

	utils.ExpectNil(err,
		"`kubectl apply` command failed: %s\n%s",
		out, utils.Err(err))

	utils.TestControlPlanePostInstall(h)
	utils.RunCheck(h, false)
}
