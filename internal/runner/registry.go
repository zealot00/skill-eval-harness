package runner

import "fmt"

// RuntimeRegistry stores skill runtimes by name.
type RuntimeRegistry map[string]SkillRuntime

var defaultRegistry = RuntimeRegistry{}

func init() {
	defaultRegistry.Register("demo-skill", NewDemoRuntime("demo-skill"))
}

// Register adds a runtime to the registry.
func (r RuntimeRegistry) Register(name string, runtime SkillRuntime) {
	r[name] = runtime
}

// Resolve returns a runtime by name.
func (r RuntimeRegistry) Resolve(name string) (SkillRuntime, error) {
	runtime, ok := r[name]
	if !ok {
		return nil, fmt.Errorf("runtime %q not registered", name)
	}

	return runtime, nil
}

// RegisterRuntime adds a runtime to the default registry.
func RegisterRuntime(name string, runtime SkillRuntime) {
	defaultRegistry.Register(name, runtime)
}

// ResolveRuntime returns a runtime from the default registry.
func ResolveRuntime(name string) (SkillRuntime, error) {
	return defaultRegistry.Resolve(name)
}
