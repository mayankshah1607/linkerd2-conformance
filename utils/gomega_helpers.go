package utils

import (
	"fmt"

	"github.com/onsi/gomega"
)

func ExpectNil(item interface{}, annotation string, args ...interface{}) {
	gomega.Expect(item).Should(gomega.BeNil(), fmt.Sprintf(annotation, args...))
}

func ExpectSubstring(item, substring, annotation string, args ...interface{}) {
	gomega.Expect(item).Should(gomega.ContainSubstring(substring), fmt.Sprintf(annotation, args...))
}

func ExpectEqual(received, expected interface{}, annotation string, args ...interface{}) {
	gomega.Expect(received).Should(gomega.Equal(expected),
		fmt.Sprintf(annotation, args...))
}

func ExpectTrue(item interface{}, annotation string, args ...interface{}) {
	gomega.Expect(item).Should(gomega.BeTrue(), fmt.Sprintf(annotation, args...))
}
