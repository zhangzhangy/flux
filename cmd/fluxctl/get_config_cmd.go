package main

import "github.com/spf13/cobra"

type serviceGetConfigOpts struct {
	*serviceOpts
	format string
}

func newServiceGetConfig(parent *serviceOpts) *serviceGetConfigOpts {
	return &serviceGetConfigOpts{serviceOpts: parent}
}

func (opts *serviceGetConfigOpts) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Get variables in config file, along with their values. If no keys specified, list all.",
		Example: makeExample(
			"fluxctl get-config [--format json | yaml]",
		),
		RunE: opts.RunE,
	}
	cmd.Flags().StringVar(&opts.format, "format", "yaml", "List variables with the given format. Options are json or yaml")
	return cmd
}

func (opts *serviceGetConfigOpts) RunE(_ *cobra.Command, args []string) error {
	encoder := yaml.NewEncoder
	switch opts.format {
	case "json":
		encoder = json.NewEncoder
	case "yaml":
		// noop
	default:
		return newUsageError("--format must be either json or yaml")
	}

	config, err := opts.Fluxd.GetConfig()
	if err != nil {
		return err
	}

	return encoder(os.Stdout).Encode(config)
}
