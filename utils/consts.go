package utils

var (
	emojivotoDeploys = map[string]int{
		"emoji":  1,
		"voting": 1,
		"web":    1,
	}

	booksappDeploys = map[string]int{
		"webapp":  3,
		"authors": 1,
		"books":   1,
	}
)

const (
	defaultNs            = "l5d-conformance"
	defaultClusterDomain = "cluster.local"
	defaultPath          = "/.linkerd2/bin/linkerd"

	versionEndpointURL = "https://versioncheck.linkerd.io/version.json"

	emojivotoNs = "emojivoto"
	booksappNs  = "booksapp"

	//TODO: move these to ConformanceTestOptions while writing Helm tests
	helmPath        = "target/helm"
	helmChart       = "charts/linkerd2"
	helmStableChart = "linkerd/linkerd2"
	helmReleaseName = ""

	// TODO: move these to config while adding tests for multicluster
	multicluster                = false
	multiclusterHelmChart       = "multicluster-helm-chart"
	multiclusterHelmReleaseName = "multicluster-helm-release"

	installEnv              = "LINKERD2_VERSION"
	configFile              = "config.yaml"
	linkerdInstallScript    = "install.sh"
	glooctlInstallScript    = "gloo_install.sh"
	linkerdInstallScriptURL = "https://run.linkerd.io/install"
	glooctlInstallScriptURL = "https://run.solo.io/gloo/install"
)
