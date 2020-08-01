package stat

import (
	"fmt"
	"strings"

	"github.com/linkerd/linkerd2-conformance/utils"
	"github.com/onsi/ginkgo"
)

func RunStatTests() bool {
	return ginkgo.Describe("stat:", func() {
		ginkgo.It("deploying sample application [emojivoto]", func() {
			utils.TestEmojivotoApp()
			utils.TestEmojivotoInject()
		})

		ginkgo.Context("running", func() {
			for _, tc := range testCases {
				tc := tc //pin
				ginkgo.It(fmt.Sprintf("`linkerd %s`", strings.Join(tc.args, " ")), func() {
					testStat(tc)
				})
			}
		})

		ginkgo.It("uninstalling sample application [emojivoto]", func() {
			utils.TestEmojivotoUninstall()
		})
	})
}
