package cmd

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/avinor/tau/internal/templates"
	"github.com/avinor/tau/pkg/config/loader"
	"github.com/avinor/tau/pkg/helpers/paths"
	"github.com/avinor/tau/pkg/helpers/ui"
	"github.com/avinor/tau/pkg/shell"
	"github.com/avinor/tau/pkg/shell/processors"
)

type destroyCmd struct {
	meta

	autoApprove bool
}

var (
	// destroyLong is long description of destroy command
	destroyLong = templates.LongDesc(`Destroy resources managed by a module. It
		can either destroy a single resource or all of them. Requires that the 
		module have been initialized first.
		`)

	// destroyExample is examples for destroy command
	destroyExample = templates.Examples(`
		# Destroy all resources from local folder
		tau destroy

		# Destroy all resources in file
		tau destroy -f module.hcl
	`)
)

// newDestroyCmd creates a new destroy command
func newDestroyCmd() *cobra.Command {
	dc := &destroyCmd{}

	destroyCmd := &cobra.Command{
		Use:                   "destroy [-f SORUCE]",
		Short:                 "Destroy tau managed infrastructure",
		Long:                  destroyLong,
		Example:               destroyExample,
		DisableFlagsInUseLine: true,
		SilenceUsage:          true,
		SilenceErrors:         true,
		Args:                  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := dc.meta.init(args); err != nil {
				return err
			}

			return dc.run(args)
		},
	}

	f := destroyCmd.Flags()
	f.BoolVar(&dc.autoApprove, "auto-approve", false, "auto approve destruction")

	dc.addMetaFlags(destroyCmd)

	return destroyCmd
}

func (dc *destroyCmd) run(args []string) error {
	// load all sources
	files, err := dc.load()
	if err != nil {
		return err
	}

	// Want to destroy them in reverse order
	for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
		files[i], files[j] = files[j], files[i]
	}

	// Verify all modules have been initialized
	if dc.meta.noAutoInit {
		if err := files.IsAllInitialized(); err != nil {
			return err
		}
	}

	for _, file := range files {
		if err := dc.runFile(file); err != nil {
			return err
		}
	}

	ui.NewLine()

	return nil
}

func (dc *destroyCmd) runFile(file *loader.ParsedFile) error {
	ui.Separator(file.Name)

	// Running prepare hook

	ui.Header("Executing prepare hooks...")

	if err := dc.Runner.Run(file, "prepare", "destroy"); err != nil {
		return err
	}

	dc.autoInit(file)

	// Resolving dependencies

	if !paths.IsFile(file.VariableFile()) {
		success, err := dc.resolveDependencies(file)
		if err != nil {
			return err
		}

		if !success {
			return nil
		}
	}

	// Executing terraform command

	ui.NewLine()
	ui.Info(color.New(color.FgGreen, color.Bold).Sprint("Tau has been successfully initialized!"))
	ui.NewLine()

	if !paths.IsFile(file.VariableFile()) {
		ui.Warn("No values file exists")
		return nil
	}

	options := &shell.Options{
		WorkingDirectory: file.ModuleDir(),
		Stdout:           shell.Processors(processors.NewUI(ui.Info)),
		Stderr:           shell.Processors(processors.NewUI(ui.Error)),
		Env:              file.Env,
	}

	extraArgs := getExtraArgs(dc.Engine.Compatibility.GetInvalidArgs("destroy")...)

	if dc.autoApprove {
		extraArgs = append(extraArgs, "-auto-approve")
	}

	if err := dc.Engine.Executor.Execute(options, "destroy", extraArgs...); err != nil {
		return err
	}

	paths.Remove(file.VariableFile())

	// Executing finish hook

	ui.Header("Executing finish hooks...")

	if err := dc.Runner.Run(file, "finish", "destroy"); err != nil {
		return err
	}

	return nil
}
