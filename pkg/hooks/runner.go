package hooks

import (
	"fmt"
	"strings"
	"sync"

	"github.com/avinor/tau/pkg/config"
	"github.com/avinor/tau/pkg/config/loader"
	"github.com/avinor/tau/pkg/getter"
	pstrings "github.com/avinor/tau/pkg/helpers/strings"
	"github.com/avinor/tau/pkg/helpers/ui"
	"github.com/avinor/tau/pkg/hooks/command"
	"github.com/avinor/tau/pkg/hooks/def"
	"github.com/go-errors/errors"
)

var (
	// noExecutorFound is returned when no executor is found among available ones
	noExecutorFound = errors.Errorf("no available executor is found for hook")
)

// Options sent to New function when making a new Runner.
type Options struct {
	Getter   *getter.Client
	CacheDir string
}

type Runner struct {
	options *Options

	// cacheLock makes sure only one executor can be generated at a time. For thread safety
	cacheLock sync.Mutex

	// cache of all created executors
	cache map[string]def.Executor

	creators []def.ExecutorCreator
}

func New(options *Options) *Runner {
	if options == nil {
		options = &Options{}
	}

	return &Runner{
		options: options,
		cache:   map[string]def.Executor{},
		creators: []def.ExecutorCreator{
			&command.Creator{},
		},
	}
}

// Run all hooks in source for a specific event. Command input can filter hooks that should only be run
// got specific terraform commands.
func (r *Runner) Run(file *loader.ParsedFile, event, command string) error {
	for _, hook := range file.Config.Hooks {
		exec, err := r.getExecutor(hook)
		if err != nil {
			return err
		}

		if !r.ShouldRun(hook, event, command) {
			ui.Debug("%s should not run for command %s", hook.Type, command)
			continue
		}

		if !exec.HasRun() || (hook.DisableCache != nil && *hook.DisableCache) {
			if err := exec.Run(); err != nil {
				return err
			}
		}

		if hook.SetEnv != nil && *hook.SetEnv {
			for key, value := range pstrings.ParseVars(exec.Output()) {
				ui.Debug("setting env %s", key)
				file.Env[key] = value
			}
		}
	}

	return nil
}

// RunAll executes the hook `event` for all files in collection
// TODO Remove when fixing output
func (r *Runner) RunAll(files loader.ParsedFileCollection, event string, command string) error {
	ui.Header(fmt.Sprintf("Executing %s hook...", event))
	for _, file := range files {
		if err := r.Run(file, event, command); err != nil {
			return err
		}
	}

	return nil
}

// ShouldRun checks if the hook should run for event and command sent as input.
// Returns true if it should continue to process hook, and false otherwise.
func (r *Runner) ShouldRun(hook *config.Hook, event, command string) bool {
	event = strings.ToLower(event)
	command = strings.ToLower(command)

	split := strings.Split(*hook.TriggerOn, ":")
	hookEvent := strings.ToLower(split[0])
	hookCommands := []string{}

	if len(split) > 1 {
		for _, cmd := range strings.Split(split[1], ",") {
			hookCommands = append(hookCommands, strings.ToLower(cmd))
		}
	}

	if hookEvent != event {
		return false
	}

	if len(hookCommands) == 0 {
		return true
	}

	for _, cmd := range hookCommands {
		if cmd == command {
			return true
		}
	}

	return false
}

func (r *Runner) getExecutor(hook *config.Hook) (def.Executor, error) {
	key := getCacheKey(hook)
	r.cacheLock.Lock()
	defer r.cacheLock.Unlock()

	if _, exists := r.cache[key]; exists {
		return r.cache[key], nil
	}

	for _, creator := range r.creators {
		if creator.CanCreate(hook) {
			r.cache[key] = creator.Create(hook)
			return r.cache[key], nil
		}
	}

	return nil, noExecutorFound
}

// getCacheKey returns a unique cache key for a given command with arguments. If disable_cache
// is set it will generate a random key to make sure it creates new instances
func getCacheKey(hook *config.Hook) string {
	var sb strings.Builder

	if hook.WorkingDir != nil && *hook.WorkingDir != "" {
		sb.WriteString(*hook.WorkingDir)
	}

	if hook.Command != nil {
		sb.WriteString(*hook.Command)
	}

	if hook.Script != nil {
		sb.WriteString(*hook.Script)
	}

	if hook.Arguments != nil {
		sb.WriteString(strings.Join(*hook.Arguments, "_"))
	}

	return sb.String()
}
