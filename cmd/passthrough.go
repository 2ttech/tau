package cmd

import (
	"github.com/avinor/tau/pkg/config"
	"github.com/avinor/tau/pkg/dir"
	"github.com/avinor/tau/pkg/shell"
	"github.com/avinor/tau/pkg/shell/processors"
	"github.com/spf13/cobra"
)

type ptCmd struct {
	meta
	command Command
}

func newPtCmd(command Command) *cobra.Command {
	pt := &ptCmd{
		command: command,
	}

	ptCmd := &cobra.Command{
		Use:   command.Use,
		Short: command.ShortDescription,
		Long:  command.LongDescription,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			if err := pt.processArgs(args); err != nil {
				return err
			}

			return pt.run(args)
		},
	}

	f := ptCmd.Flags()
	pt.addMetaFlags(f)

	return ptCmd
}

func (pt *ptCmd) run(args []string) error {
	loaded, err := config.LoadSourcesFile(pt.TempDir)
	if err != nil {
		return err
	}

	for _, source := range loaded {
		moduleDir := dir.Module(pt.TempDir, source.Name)

		options := &shell.Options{
			WorkingDirectory: moduleDir,
			Stdout:           shell.Processors(new(processors.Log)),
			Stderr:           shell.Processors(new(processors.Log)),
		}

		extraArgs := getExtraArgs(args, pt.Engine.Compatibility.GetInvalidArgs(pt.command.Use)...)
		if err := pt.Engine.Executor.Execute(options, pt.command.Use, extraArgs...); err != nil {
			return err
		}
	}

	return nil
}
