package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/linkerd/linkerd2/testutil"
	"gopkg.in/yaml.v2"
)

var (
	defaultNs            = "l5d-conformance"
	defaultClusterDomain = "cluster.local"
	defaultVersion       = "stable-2.8.0"
	defaultPath          = "/.linkerd2/bin/linkerd"

	versionEndpointURL = "https://versioncheck.linkerd.io/version.json"
)

// Inject holds the inject test configuration
type Inject struct {
	Skip  bool `yaml:"skip,omitempty"`
	Clean bool `yaml:"clean,omitempty"` // deletes all resources created while testing
}

type Lifecycle struct {
	Skip               bool   `yaml:"skip,omitempty"`
	UpgradeFromVersion string `yaml:"upgradeFromVersion,omitempty"`
	Reinstall          bool   `yaml:"reinstall,omitempty"`
	Uninstall          bool   `yaml:"uninstall,omitempty"`
}

type ControlPlaneConfig struct {
	HA     bool                   `yaml:"ha,omitempty"`
	Flags  []string               `yaml:"flags,omitempty"`
	AddOns map[string]interface{} `yaml:"addOns,omitempty"`
}

type ControlPlane struct {
	Namespace          string `yaml:"namespace,omitempty"`
	ControlPlaneConfig `yaml:"config,omitempty"`
}

// ConformanceTestOptions holds the values fed from the test config file
type ConformanceTestOptions struct {
	LinkerdVersion    string `yaml:"linkerdVersion,omitempty"`
	LinkerdBinaryPath string `yaml:"linkerdPath,omitempty"`
	ClusterDomain     string `yaml:"clusterDomain,omitempty"`
	K8sContext        string `yaml:"k8sContext,omitempty"`
	ExternalIssuer    bool   `yaml:"externalIssuer,omitempty"`
	ControlPlane      `yaml:"controlPlane"`
	Lifecycle         `yaml:"lifecycle,omitempty"`
	Inject            `yaml:"inject"`

	// TODO: Add fields for test specific configurations
	// TODO: Add fields for Helm tests
}

func getLatestStableVersion() (string, error) {

	var versionResp map[string]string

	req, err := http.NewRequest("GET", versionEndpointURL, nil)
	if err != nil {
		return "", err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(body, &versionResp); err != nil {
		return "", err
	}

	return versionResp["stable"], nil
}

func getDefaultLinkerdPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/.linkerd2/bin/linkerd", home), nil
}

func initK8sHelper(context string, retryFor func(time.Duration, func() error) error) (*testutil.KubernetesHelper, error) {
	k8sHelper, err := testutil.NewKubernetesHelper(context, retryFor)
	if err != nil {
		return nil, err
	}

	return k8sHelper, nil
}

func (options *ConformanceTestOptions) parse() error {
	if options.LinkerdVersion == "" {
		var version string
		var err error

		if version, err = getLatestStableVersion(); err != nil {
			fmt.Printf("error fetching latest version: %s\n", err)
			version = defaultVersion
		}

		fmt.Printf("Unspecified linkerd2 version - using default value \"%s\"\n", version)
		options.LinkerdVersion = version
	}

	if options.ControlPlane.Namespace == "" {
		fmt.Printf("Unspecified linkerd2 control plane namespace - use default value \"%s\"\n", defaultNs)
		options.ControlPlane.Namespace = defaultNs
	}

	if options.ClusterDomain == "" {
		fmt.Printf("Unspecified cluster domain - using default value \"%s\"\n", defaultClusterDomain)
		options.ClusterDomain = defaultClusterDomain
	}

	if options.LinkerdBinaryPath == "" {
		path, err := getDefaultLinkerdPath()
		if err != nil {
			return err
		}
		fmt.Printf("Unspecified path to linkerd2 binary - using default value \"%s\"\n", path)
		options.LinkerdBinaryPath = path
	}

	if !options.SingleControlPlane() && options.Lifecycle.Uninstall {
		fmt.Println("'globalControlPlane.uninstall' will be ignored as globalControlPlane is disabled")
		options.Lifecycle.Uninstall = false
	}

	if options.SingleControlPlane() && options.SkipLifecycle() {
		return errors.New("Cannot skip lifecycle tests when 'install.globalControlPlane.enable' is set to \"true\"")
	}

	if options.Lifecycle.UpgradeFromVersion != "" && options.SkipLifecycle() {
		return errors.New("cannot skip lifecycle tests when 'install.upgradeFromVersion' is set - either enable install tests, or omit 'install.upgradeFromVersion'")
	}
	return nil
}

func (options *ConformanceTestOptions) initNewTestHelperFromOptions() (*testutil.TestHelper, error) {
	var (
		//TODO: move these to ConformanceTestOptions while writing Helm tests
		helmPath        = "target/helm"
		helmChart       = "charts/linkerd2"
		helmStableChart = "linkerd/linkerd2"
		helmReleaseName = ""

		// TODO: move these to config while adding tests for multicluster
		multicluster                = false
		multiclusterHelmChart       = "multicluster-helm-chart"
		multiclusterHelmReleaseName = "multicluster-helm-release"
	)

	httpClient := http.Client{
		Timeout: 10 * time.Second,
	}

	helper := testutil.NewGenericTestHelper(
		options.LinkerdBinaryPath,
		options.LinkerdVersion,
		options.ControlPlane.Namespace,
		options.Lifecycle.UpgradeFromVersion,
		options.ClusterDomain,
		helmPath,
		helmChart,
		helmStableChart,
		helmReleaseName,
		multiclusterHelmReleaseName,
		multiclusterHelmChart,
		options.ExternalIssuer,
		multicluster,
		options.Lifecycle.Uninstall,
		httpClient,
		testutil.KubernetesHelper{},
	)

	k8sHelper, err := initK8sHelper(options.K8sContext, helper.RetryFor)
	if err != nil {
		return nil, fmt.Errorf("error initializing k8s helper: %s", err)
	}

	helper.KubernetesHelper = *k8sHelper
	return helper, nil
}

// The below defined methods on *ConformanceTestOptions will return
// test specific configuration.

// However, *TestHelper must be used for obtaining the following fields:
// - LinkerdVersion -> TestHelper.GetVersion()
// - LinkerdNamespace -> TestHelper.GetTestNamespace()
// - GlobalControlPlane.Uninstall -> TestHelper.Uninstall()
// - Install.UpgradeFromVersion -> TestHelper.UpgradeFromVersion()
// - ExternalIssuer -> TestHelper.ExternalIssuer()
// - ClusterDomain -> TestHelper.ClusterDomain()

// GetLinkerdPath returns the path where Linkerd binary will be installed
func (options *ConformanceTestOptions) GetLinkerdPath() string {
	return options.LinkerdBinaryPath
}

// SingleControlPlane determines if a singl CP must be used throughout
func (options *ConformanceTestOptions) SingleControlPlane() bool {
	return !options.Lifecycle.Reinstall
}

// HA determines if a high-availability control-plane must be used
func (options *ConformanceTestOptions) HA() bool {
	return options.ControlPlane.ControlPlaneConfig.HA
}

// SkipInstall determines if install tests must be skipped
func (options *ConformanceTestOptions) SkipLifecycle() bool {
	return !options.SingleControlPlane() && options.Lifecycle.Skip
}

// CleanInject determines if resources created during inject test must be removed
func (options *ConformanceTestOptions) CleanInject() bool {
	return options.Inject.Clean
}

// SkipInject determines if inject test must be skipped
func (options *ConformanceTestOptions) SkipInject() bool {
	return options.Inject.Skip
}

// GetAddons returns the add-on config
func (options *ConformanceTestOptions) GetAddons() map[string]interface{} {
	return options.ControlPlane.ControlPlaneConfig.AddOns
}

// GetAddOnsYAML marshals the add-on config to a YAML and returns the byte slice and error
func (options *ConformanceTestOptions) GetAddOnsYAML() (out []byte, err error) {
	return yaml.Marshal(options.GetAddons())
}

// GetInstallFlags returns the flags set by the user for running `linkerd install`
func (options *ConformanceTestOptions) GetInstallFlags() []string {
	return options.ControlPlane.ControlPlaneConfig.Flags
}
