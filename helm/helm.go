package helm

import (
	"errors"
	"fmt"
	"github.com/random-dwi/helm-doc/output"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"
	"os"
	"path/filepath"
	"strings"
)

var helmHome = helmpath.Home(os.Getenv("HELM_HOME"))

var settings = helm_env.EnvSettings{
	Home: helmHome,
}

// locateChartPath looks for a chart directory in known places, and returns either the full path or an error.
//
// This does not ensure that the chart is well-formed; only that the requested filename exists.
//
// Order of resolution:
// - current working directory
// - if path is absolute or begins with '.', error out here
// - chart repos in $HELM_HOME
// - URL
//
// If 'verify' is true, this will attempt to also verify the chart.
func LocateChartPath(repoURL, username, password, name, version string, verify bool, keyring,
	certFile, keyFile, caFile string) (string, error) {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if fi, err := os.Stat(name); err == nil {
		abs, err := filepath.Abs(name)
		if err != nil {
			return abs, err
		}
		if verify {
			if fi.IsDir() {
				return "", errors.New("cannot verify a directory")
			}
			if _, err := downloader.VerifyChart(abs, keyring); err != nil {
				return "", err
			}
		}
		return abs, nil
	}
	if filepath.IsAbs(name) || strings.HasPrefix(name, ".") {
		return name, fmt.Errorf("path %q not found", name)
	}

	crepo := filepath.Join(helmHome.Repository(), name)
	if _, err := os.Stat(crepo); err == nil {
		return filepath.Abs(crepo)
	}

	dl := downloader.ChartDownloader{
		HelmHome: helmHome,
		Out:      os.Stdout,
		Keyring:  keyring,
		Getters:  getter.All(settings),
		Username: username,
		Password: password,
	}
	if verify {
		dl.Verify = downloader.VerifyAlways
	}
	if repoURL != "" {
		chartURL, err := repo.FindChartInAuthRepoURL(repoURL, username, password, name, version,
			certFile, keyFile, caFile, getter.All(settings))
		if err != nil {
			return "", err
		}
		name = chartURL
	}

	if _, err := os.Stat(helmHome.Archive()); os.IsNotExist(err) {
		os.MkdirAll(helmHome.Archive(), 0744)
	}

	filename, _, err := dl.DownloadTo(name, version, helmHome.Archive())
	if err == nil {
		lname, err := filepath.Abs(filename)
		if err != nil {
			return filename, err
		}
		output.Debugf("Fetched %s to %s", name, filename)
		return lname, nil
	} else if settings.Debug {
		return filename, err
	}

	return filename, fmt.Errorf("failed to download %q (hint: running `helm repo update` may help)", name)
}
