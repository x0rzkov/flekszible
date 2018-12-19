package processor

import "github.com/elek/flekszible/pkg/data"

type Image struct {
	DefaultProcessor
	Image string
}

func (imageSet *Image) BeforeResource(resource *data.Resource) {
	resource.Content.Accept(&data.Set{Path: data.NewPath("spec", "template", "spec", "(initC|c)ontainers", ".*", "image"), NewValue: imageSet.Image})
}
func init() {
	prototype := Image{}
	ProcessorTypeRegistry.Add(&prototype)
}