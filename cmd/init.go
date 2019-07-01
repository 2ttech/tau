package cmd

import (
	"net/http"
	"os"
	"time"

	"github.com/apex/log"
	"github.com/avinor/tau/internal/templates"
	"github.com/avinor/tau/pkg/config"
	"github.com/avinor/tau/pkg/getter"
	"github.com/avinor/tau/pkg/paths"
	"github.com/avinor/tau/pkg/shell"
	"github.com/avinor/tau/pkg/shell/processors"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type initCmd struct {
	meta

	getter *getter.Client
	loader *config.Loader

	maxDependencyDepth int
	purge              bool
	source             string
	sourceVersion      string
}

var (
	sourceMustBeAFile = errors.Errorf("file cannot be a directory when source is overridden")

	initLong = templates.LongDesc(`Initialize tau working folder based on SOURCE argument.
		SOURCE can either be a single file or a folder. If it is a folder it will initialize
		all modules in the folder, ordering them by dependencies.
		`)

	initExample = templates.Examples(`
		# Initialize a single module
		tau init module.hcl

		# Initialize current folder
		tau init

		# Initialize a module and send additional argument to terraform
		tau init module.hcl -input=false
	`)
)

func newInitCmd() *cobra.Command {
	ic := &initCmd{}

	initCmd := &cobra.Command{
		Use:                   "init [-f SOURCE]",
		Short:                 "Initialize a tau working directory",
		Long:                  initLong,
		Example:               initExample,
		DisableFlagsInUseLine: true,
		SilenceUsage:          true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ic.meta.processArgs(args); err != nil {
				return err
			}

			if err := ic.processArgs(args); err != nil {
				return err
			}

			ic.init()

			return ic.run(args)
		},
	}

	f := initCmd.Flags()
	f.IntVar(&ic.maxDependencyDepth, "max-dependency-depth", 1, "defines max dependency depth when traversing dependencies") //nolint:lll
	f.BoolVar(&ic.purge, "purge", false, "purge temporary folder before init")
	f.StringVar(&ic.source, "source", "", "override module source location")
	f.StringVar(&ic.sourceVersion, "source-version", "", "override module source version, only valid together with source override")

	ic.addMetaFlags(initCmd)

	return initCmd
}

func (ic *initCmd) init() {
	{
		timeout := time.Duration(ic.timeout) * time.Second

		log.Debugf("- Http timeout: %s", timeout)

		options := &getter.Options{
			HttpClient: &http.Client{
				Timeout: timeout,
			},
			WorkingDirectory: paths.WorkingDir,
		}

		ic.getter = getter.New(options)
	}

	{
		options := &config.Options{
			WorkingDirectory: paths.WorkingDir,
			TempDirectory:    ic.TempDir,
			MaxDepth:         ic.maxDependencyDepth,
		}

		ic.loader = config.NewLoader(options)
	}
}

func (ic *initCmd) processArgs(args []string) error {
	if ic.source != "" {
		fi, err := os.Stat(ic.file)
		if err != nil {
			return err
		}

		if fi.IsDir() {
			return sourceMustBeAFile
		}
	}

	return nil
}

func (ic *initCmd) run(args []string) error {
	if ic.purge {
		log.Debug("Purging temporary folder")
		log.Debug("")
		paths.Remove(ic.TempDir)
	}

	loaded, err := ic.loader.Load(ic.file)
	if err != nil {
		return err
	}

	log.Info("")

	if len(loaded) == 0 {
		log.Warn("No sources found in path")
		return nil
	}

	log.Info(color.New(color.Bold).Sprint("Loading modules..."))
	for _, source := range loaded {
		module := source.Config.Module
		moduleDir := paths.ModuleDir(ic.TempDir, source.Name)

		if module == nil {
			log.WithField("file", source.Name).Fatal("No module defined in source")
			continue
		}

		source := module.Source
		version := module.Version

		if ic.source != "" {
			source = ic.source
			version = &ic.sourceVersion
		}

		if err := ic.getter.Get(source, moduleDir, version); err != nil {
			return err
		}
	}
	log.Info("")

	log.Info(color.New(color.Bold).Sprint("Resolving dependencies..."))
	for _, source := range loaded {
		if source.Config.Inputs == nil {
			continue
		}

		moduleDir := paths.ModuleDir(ic.TempDir, source.Name)
		depsDir := paths.DependencyDir(ic.TempDir, source.Name)

		vars, err := ic.Engine.ResolveDependencies(source, depsDir)
		if err != nil {
			return err
		}

		if err := ic.Engine.WriteInputVariables(source, moduleDir, vars); err != nil {
			return err
		}
	}
	log.Info("")

	for _, source := range loaded {
		moduleDir := paths.ModuleDir(ic.TempDir, source.Name)

		if err := ic.Engine.CreateOverrides(source, moduleDir); err != nil {
			return err
		}

		options := &shell.Options{
			WorkingDirectory: moduleDir,
			Stdout:           shell.Processors(new(processors.Log)),
			Stderr:           shell.Processors(new(processors.Log)),
			Env:              source.Env,
		}

		log.Info("------------------------------------------------------------------------")

		extraArgs := getExtraArgs(args, ic.Engine.Compatibility.GetInvalidArgs("init")...)
		if err := ic.Engine.Executor.Execute(options, "init", extraArgs...); err != nil {
			return err
		}
	}

	return config.SaveCheckpoint(loaded, ic.TempDir)
}
