package main

import (
	"github.com/random-dwi/helm-doc/cmd"
	"github.com/random-dwi/helm-doc/output"
	"os"
)

func main() {

	ioStreams := output.DefaultIOStreams()

	if err := cmd.HelmDocCommand(ioStreams).Execute(); err != nil {
		output.Warnf("%v", err)
		os.Exit(1)
	}
}
