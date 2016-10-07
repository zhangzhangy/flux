package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type serviceSetConfigOpts struct {
	*serviceOpts
	file string
}

func newServiceSetConfig(parent *serviceOpts) *serviceSetConfigOpts {
	return &serviceSetConfigOpts{serviceOpts: parent}
}

func (opts *serviceSetConfigOpts) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Set key(s) in config file.",
		Example: makeExample(
			"fluxctl set-config repo.url=git@github.com:weaveworks/helloworld repo.path=k8s/local",
			"fluxctl set-config --file fluxconfig.yaml",
		),
		RunE: opts.RunE,
	}
	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "Use the given config file instead of the existing one. All existing values will be removed.")
	return cmd
}

func (opts *serviceSetConfigOpts) RunE(_ *cobra.Command, args []string) error {
	if len(args) > 0 {
		return errorWantedNoArgs
	}
	if opts.file == "" {
		return newUsageError("-f, --file is required")
	}

	var newConfig flux.Config
	f, err := os.Open(opts.file)
	if err != nil {
		return errors.Wrapf(err, "opening config file %s", opts.file)
	}
	ext := filepath.Ext(opts.file)
	switch ext {
	case ".json":
		if err := json.NewDecoder(f).Decode(&newConfig); err != nil {
			return errors.Wrapf(err, "parsing json file %s", opts.file)
		}
	case ".yaml", ".yml":
		fallthrough
	default:
		if err := yaml.NewDecoder(f).Decode(&newConfig); err != nil {
			return errors.Wrapf(err, "parsing yaml file %s", opts.file)
		}
	}

	return opts.Fluxd.SetConfig(newConfig)
}
