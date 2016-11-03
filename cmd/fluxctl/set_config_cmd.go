package main

import (
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/weaveworks/flux"
)

type setConfigOpts struct {
	*rootOpts
	file   string
	update flux.ConfigUpdate
}

func newSetConfig(parent *rootOpts) *setConfigOpts {
	return &setConfigOpts{rootOpts: parent}
}

func (opts *setConfigOpts) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-config",
		Short: "set configuration values for an instance",
		Example: makeExample(
			"fluxctl config --file=./dev/flux-conf.yaml",
		),
		RunE: opts.RunE,
	}
	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "A file to upload as configuration; this will overwrite all values.")
	cmd.Flags().BoolVar(&opts.update.GenerateKey, "generate-key", false, "Generate a key pair and return the public key for use as a deploy key.")
	return cmd
}

func (opts *setConfigOpts) RunE(_ *cobra.Command, args []string) error {
	if len(args) > 0 {
		return errorWantedNoArgs
	}

	if opts.file == "" {
		return newUsageError("-f, --file is required")
	}

	var config flux.InstanceConfig

	bytes, err := ioutil.ReadFile(opts.file)
	if err == nil {
		err = yaml.Unmarshal(bytes, &config)
	}
	if err != nil {
		return errors.Wrapf(err, "reading config from file")
	}

	opts.update.Config = config
	result, err := opts.API.SetConfig(noInstanceID, opts.update)
	if err != nil {
		return err
	}
	return outputResult(opts.update, result)
}

func outputResult(sent flux.ConfigUpdate, result flux.InstanceConfig) error {
	if sent.GenerateKey {
		if result.Git.PublicKey == "" {
			return errors.New("Key generation requested, but no public key returned")
		}
		fmt.Println(result.Git.PublicKey)
	}
	return nil
}
