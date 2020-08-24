module github.com/linkerd/linkerd2-conformance

go 1.14

require (
	github.com/linkerd/linkerd2 v0.5.1-0.20200729024817-96f662dfacf0
	github.com/onsi/ginkgo v1.13.0
	github.com/onsi/gomega v1.10.1
	github.com/wercker/stern v0.0.0-20190705090245-4fa46dd6987f
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/apimachinery v0.17.4
	sigs.k8s.io/yaml v1.2.0
)

replace github.com/wercker/stern => github.com/linkerd/stern v0.0.0-20200331220320-37779ceb2c32
