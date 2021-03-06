package processor

import (
	"github.com/elek/flekszible/api/v2/data"
	"github.com/elek/flekszible/api/v2/yaml"
)

type Env struct {
	DefaultProcessor
	Envs    map[string]string
	Trigger Trigger
}

func (env *Env) BeforeResource(resource *data.Resource) error {
	if env.Trigger.active(resource) {
		smartGet := data.SmartGetAll{Path: data.NewPath("spec", "template", "spec", ".*ontainers", ".*", "env")}
		resource.Content.Accept(&smartGet)
		for _, result := range smartGet.Result {
			envs := result.Value.(*data.ListNode)
			for key, value := range env.Envs {
				envEntry := envs.CreateMap()
				envEntry.PutValue("name", key)
				envEntry.PutValue("value", value)
			}
		}
	}
	return nil
}
func ActivateEnv(registry *ProcessorTypes) {
	registry.Add(ProcessorDefinition{
		Metadata: ProcessorMetadata{
			Name:        "Env",
			Description: "Add environment variables to Statefulset/Deployment/...",
			Doc:         "Use any KEY=value",
		},
		Factory: func(config *yaml.MapSlice) (Processor, error) {
			envProc := &Env{}
			envProc.Envs = make(map[string]string)
			for _, item := range *config {
				if item.Key.(string) != "type" && item.Key.(string) != "trigger" {
					envProc.Envs[item.Key.(string)] = item.Value.(string)
				}
			}
			cleanSettings := yaml.MapSlice{}
			if triggerConfig, found := config.Get("trigger"); found {
				cleanSettings.Put("trigger", triggerConfig)
			}
			proc, err := configureProcessorFromYamlFragment(envProc, &cleanSettings)
			return proc, err
		},
	})
}
