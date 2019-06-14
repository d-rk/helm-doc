package cmd

import (
	"errors"
	"fmt"
	"github.com/random-dwi/helm-doc/generator"
	"github.com/random-dwi/helm-doc/helm"
	"github.com/random-dwi/helm-doc/output"
	"github.com/random-dwi/helm-doc/writer"
	"github.com/spf13/cobra"
	"k8s.io/helm/pkg/chartutil"
	"log"
	"os"
)

var version string
var buildTime string
var gitCommit string

var rootCmd = &cobra.Command{
	Use:   "doc [flags] CHART",
	Short: fmt.Sprintf("generate doc for a helm chart"),
	Long:  fmt.Sprintf("helm plugin to generate documentation for helm charts.\nversion: %s buildTime: %s gitCommit: %s", version, buildTime, gitCommit),
	RunE:  run,
}

var flags generator.CommandFlags

func HelmDocCommand(streams output.IOStreams) *cobra.Command {
	rootCmd.SetOutput(streams.Out)
	return rootCmd
}

func defaultKeyring() string {
	return os.ExpandEnv("$HOME/.gnupg/pubring.gpg")
}

func init() {

	f := rootCmd.Flags()
	f.BoolVarP(&flags.VerifyExamples, "verify-examples", "", true, "verify presence of examples for configs without default value")
	f.BoolVarP(&flags.VerifyValues, "verify-values", "", true, "verify all default values are documented")
	f.BoolVarP(&flags.VerifyDependencies, "verify-dependencies", "", false, "verify dependencies are documented")
	f.StringVar(&flags.Version, "version", "", "Specify the exact chart version to use. If this is not specified, the latest version is used")
	f.StringVar(&flags.RepoURL, "repo", "", "Chart repository url where to locate the requested chart")
	f.StringVar(&flags.Username, "username", "", "Chart repository username where to locate the requested chart")
	f.StringVar(&flags.Password, "password", "", "Chart repository password where to locate the requested chart")
	f.StringVar(&flags.Keyring, "keyring", defaultKeyring(), "Location of public keys used for verification")
	f.StringVar(&flags.CertFile, "cert-file", "", "Identify HTTPS client using this SSL certificate file")
	f.StringVar(&flags.KeyFile, "key-file", "", "Identify HTTPS client using this SSL key file")
	f.StringVar(&flags.CaFile, "ca-file", "", "Verify certificates of HTTPS-enabled servers using this CA bundle")
	f.BoolVar(&flags.Verify, "verify", false, "Verify the package before using it")
	f.BoolVar(&flags.Devel, "devel", false, "Use development versions, too. Equivalent to version '>0.0.0-0'. If --version is set, this is ignored.")

	if os.Getenv("HELM_DEBUG") == "1" {
		flags.Verbose = true
	}
}

func run(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("chart is required")
	}

	if flags.Verbose {
		output.DebugLogger = log.New(os.Stderr, "[doc] ", log.LstdFlags)
	}

	output.Debugf("helm home: %s", os.Getenv("HELM_HOME"))

	output.Debugf("Original chart version: %q", flags.Version)
	if flags.Version == "" && flags.Devel {
		output.Debugf("setting version to >0.0.0-0")
		flags.Version = ">0.0.0-0"
	}

	cp, err := helm.LocateChartPath(flags.RepoURL, flags.Username, flags.Password, args[0], flags.Version, flags.Verify, flags.Keyring,
		flags.CertFile, flags.KeyFile, flags.CaFile)
	if err != nil {
		return err
	}
	var chartPath = cp

	output.Debugf("ChartPath is: %s", chartPath)

	c, err := chartutil.Load(chartPath)
	if err != nil {
		return err
	}

	var gen writer.DocumentationWriter = writer.NewMarkdownWriter(os.Stdout)

	var dependencyNames []string

	for _, dependency := range c.Dependencies {
		dependencyNames = append(dependencyNames, dependency.Metadata.Name)
	}

	output.Debugf("generating docs for %s:%s", c.Metadata.Name, c.Metadata.Version)
	docs, err := generator.GenerateDocs(c, dependencyNames, flags)

	gen.WriteMetaData(c.Metadata, 1)
	gen.WriteDocs(docs)

	if len(c.Dependencies) > 0 {
		gen.WriteChapter("Dependencies", 2)
	}

	for _, dependency := range c.Dependencies {
		output.Debugf("generating docs for %s:%s", dependency.Metadata.Name, dependency.Metadata.Version)
		docs, err := generator.GenerateDocs(dependency, nil, flags)

		if err != nil {
			if flags.VerifyDependencies {
				output.Failf("%v", err)
			} else {
				output.Warnf("%v", err)
			}
		} else {
			gen.WriteMetaData(dependency.Metadata, 3)
			gen.WriteDocs(docs)
		}
	}

	return nil
}
