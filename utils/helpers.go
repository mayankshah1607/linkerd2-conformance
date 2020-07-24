package utils

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"github.com/linkerd/linkerd2/testutil"
	"gopkg.in/yaml.v2"
)

var (
	testHelper *testutil.TestHelper
	testConfig *ConformanceTestOptions
)

// InitNewHelperAndConfig returns an instance of testutil.TestHelper and ConformanceTestOptions
func initNewHelperAndConfig() error {

	var err error
	testConfig = &ConformanceTestOptions{}

	if fileExists(configFile) {
		yamlFile, err := ioutil.ReadFile(configFile)
		if err != nil {
			return err
		}

		if err := yaml.UnmarshalStrict(yamlFile, &testConfig); err != nil {
			return fmt.Errorf("failed to parse YAML: %s", err.Error())
		}

	}

	if err := testConfig.parse(); err != nil {
		return err
	}

	testHelper, err = testConfig.initNewTestHelperFromOptions()
	if err != nil {
		return err
	}

	return err
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func fetchInstallScript(installScriptURL string) ([]byte, error) {

	req, err := http.NewRequest("GET", installScriptURL, nil)
	if err != nil {
		return []byte{}, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return []byte{}, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return []byte{}, err
	}

	return body, nil
}

func createFileWithContent(data []byte, path string) error {
	file, err := os.Create(path)
	defer file.Close()

	if _, err = file.Write(data); err != nil {
		return err
	}
	return nil
}

// InstallGlooctlBinary installs `glooctl`
func InstallGlooctlBinary() error {
	script, err := fetchInstallScript(glooctlInstallScriptURL)
	if err != nil {
		return err
	}

	if err = createFileWithContent(script, glooctlInstallScript); err != nil {
		return err
	}

	cmd := exec.Command("/bin/bash", glooctlInstallScript)
	if err := cmd.Run(); err != nil {
		return err
	}
	if err := os.Remove(glooctlInstallScript); err != nil {
		return err
	}

	return nil
}

func getDefaultGlooPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s", home, "/.gloo/bin"), nil
}

// GlooctlRun runs a `glooctl` command
func GlooctlRun(arg ...string) (string, error) {
	path, err := getDefaultGlooPath()
	if err != nil {
		return "", err
	}

	glooctl := fmt.Sprintf("%s/%s", path, "glooctl")
	cmd := exec.Command(glooctl, arg...)

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// InstallLinkerdBinary installs a linkerd2 binary of the given version
func InstallLinkerdBinary(linkerd, version string, force bool, verbose bool) error {
	if fileExists(linkerd) && !force {
		fmt.Printf("linkerd2 binary exists in \"%s\"- skipping installation\n", linkerd)
		return nil
	}

	script, err := fetchInstallScript(linkerdInstallScriptURL)
	if err != nil {
		return err
	}

	if err = createFileWithContent(script, linkerdInstallScript); err != nil {
		return err
	}

	os.Setenv(installEnv, version)

	cmd := exec.Command("/bin/sh", linkerdInstallScript)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return err
	}

	if err = os.Remove(linkerdInstallScript); err != nil {
		return err
	}

	return nil
}

func indexOf(arr []string, item string) int {
	for i, v := range arr {
		if v == item {
			return i
		}
	}
	return -1
}

// Err returns err.Error() string
// if err is not nil
// This helper is meant to be used with
// gomega.Should() to annotate failures
// without causing runtime errors when
// returned errors are nil
func Err(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

// GetHelperAndConfig returns a reference to the initialized `testHelper` and `testConfig`
func GetHelperAndConfig() (*testutil.TestHelper, *ConformanceTestOptions) {
	return testHelper, testConfig
}

func init() {
	err := initNewHelperAndConfig()
	if err != nil {
		fmt.Printf("failed to initialize test helper or config: %s", err.Error())
		os.Exit(1)
	}
}
