package processor

import (
	"github.com/elek/flekszible/pkg/data"
	"github.com/elek/flekszible/pkg/yaml"
)

type Processor interface {
	data.Visitor
	Before(ctx *RenderContext, resources []*data.Resource)
	After(ctx *RenderContext, resources []*data.Resource)

	BeforeResource(*data.Resource)
	AfterResource(*data.Resource)
	GetScope() string
	SetScope(scope string)
}

type DefaultProcessor struct {
	data.DefaultVisitor
	Type            string
	Scope           string
	File            string
	CurrentResource *data.Resource
}

func (processor *DefaultProcessor) Before(ctx *RenderContext, resources []*data.Resource) {}
func (processor *DefaultProcessor) After(ctx *RenderContext, resources []*data.Resource)  {}
func (processor *DefaultProcessor) GetScope() string {
	return processor.Scope
}
func (processor *DefaultProcessor) SetScope(scope string) {
	processor.Scope = scope
}
func (p *DefaultProcessor) BeforeResource(resource *data.Resource) {
	p.CurrentResource = resource
}

func (p *DefaultProcessor) AfterResource(*data.Resource) {
	p.CurrentResource = nil
}

func configureProcessorFromYamlFragment(processor Processor, config *yaml.MapSlice) (Processor, error) {
	processorConfigYaml, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(processorConfigYaml, processor)
	if err != nil {
		return nil, err
	}
	return processor, nil
}