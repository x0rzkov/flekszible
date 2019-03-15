package processor

import (
	"github.com/elek/flekszible/pkg/data"
	"github.com/elek/flekszible/pkg/yaml"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

type K8sWriter struct {
	DefaultProcessor
	arrayParent       bool
	mapIndex          int
	started           bool
	resourceOutputDir string
	output            io.Writer
	file              *os.File
}

func (writer *K8sWriter) Before(ctx *RenderContext, resources []*data.Resource) {
	writer.resourceOutputDir = ctx.OutputDir
}
func (writer *K8sWriter) createOutputPath(outputDir, name, kind string, destination string) string {
	fileName := name + "-" + strings.ToLower(kind) + ".yaml"
	return path.Join(outputDir, destination, fileName)

}

func (writer *K8sWriter) BeforeResource(resource *data.Resource) {
	writer.started = false
	outputDir := writer.resourceOutputDir
	if outputDir == "-" {
		writer.output = os.Stderr
	} else {

		outputFile := writer.createOutputPath(outputDir, resource.Name(), resource.Kind(), resource.Destination)
		err := os.MkdirAll(path.Dir(outputFile), os.ModePerm)
		if err != nil {
			panic(err)
		}

		content, err := resource.Content.ToString()
		if err != nil {
			panic(err);
		}

		err = ioutil.WriteFile(outputFile, []byte(content), 0655)
		if err != nil {
			panic(err);
		}
	}
}

func CreateStdK8sWriter() *K8sWriter {
	writer := K8sWriter{
		resourceOutputDir: "-",
	}
	return &writer
}

func init() {
	ProcessorTypeRegistry.Add(ProcessorDefinition{
		Metadata: ProcessorMetadata{
			Name:        "K8sWriter",
			Description: "Internal transformation to print out k8s resources as yaml",
		},
		Factory: func(config *yaml.MapSlice) (Processor, error) {
			return configureProcessorFromYamlFragment(&K8sWriter{}, config)
		},
	})
}
