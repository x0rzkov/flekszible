package pkg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/brettski/go-termtables"
	"github.com/elek/flekszible/api/data"
	"github.com/elek/flekszible/api/processor"
	"github.com/elek/flekszible/api/yaml"
	"github.com/hashicorp/go-getter"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func ListResources(context *processor.RenderContext) {
	err := context.Init()
	if err != nil {
		panic(err)
	}
	table := termtables.CreateTable()
	table.AddHeaders("name", "kind")
	nodes := context.ListResourceNodes()
	for _, node := range nodes {
		for _, resource := range node.Resources {
			table.AddRow(resource.Name(), resource.Kind())
		}
	}
	fmt.Println("Detected resources:")
	fmt.Println(table.Render())
}

func PrintTree(node *processor.ResourceNode, prefix string) {
	fmt.Println(prefix + ">>> " + node.Dir)
	if len(node.Resources) > 0 {
		fmt.Println(prefix + "  RESOURCES:")
		for _, resource := range node.Resources {
			fmt.Println(prefix + "    " + resource.Name() + "/" + resource.Kind())
		}
	}
	if len(node.ProcessorRepository.Processors) > 0 {

		fmt.Println(prefix + "  TRANSFORMATIONS:")
		for _, trafo := range node.ProcessorRepository.Processors {
			fmt.Println(prefix + "    " + trafo.ToString())
		}
	}

	if len(node.Definitions) > 0 {
		fmt.Println(prefix + "  DEFINITIONS:")
		for _, def := range node.Definitions {
			fmt.Println(prefix + "    " + def)
		}
	}

	for _, child := range node.Children {
		PrintTree(child, "      ")
	}
	node.LoadDefinitions()

}
func Tree(context *processor.RenderContext) {

	PrintTree(context.RootResource, "")
}

func ListProcessor(context *processor.RenderContext) {
	err := context.Init()
	if err != nil {
		panic(err)
	}
	table := termtables.CreateTable()
	table.AddHeaders("name", "description")
	definitionNames := make([]string, 0)
	for definition := range processor.ProcessorTypeRegistry.TypeMap {
		definitionNames = append(definitionNames, definition)
	}
	sort.Strings(definitionNames)

	for _, definitionName := range definitionNames {
		definition := processor.ProcessorTypeRegistry.TypeMap[definitionName]
		table.AddRow(definitionName, definition.Metadata.Description)
	}
	fmt.Println(table.Render())

}

func ShowProcessor(context *processor.RenderContext, command string) {
	err := context.Init()
	if err != nil {
		panic(err)
	}

	if procDefinition, found := processor.ProcessorTypeRegistry.TypeMap[strings.ToLower(command)]; found {
		fmt.Println("")
		fmt.Println("### " + command)
		fmt.Println()
		fmt.Println(procDefinition.Metadata.Description)
		fmt.Println()
		fmt.Println("#### Parameters")
		fmt.Println("")
		table := termtables.CreateTable()
		table.AddHeaders("name", "default", "description")
		for _, parameter := range procDefinition.Metadata.Parameter {
			table.AddRow(parameter.Name, parameter.Default, parameter.Description)
		}
		fmt.Println(table.Render())
		fmt.Println()
		fmt.Println(procDefinition.Metadata.Doc)

	} else {
		fmt.Println("No such processor definition: " + command)
	}

}

func listUniqSources(context *processor.RenderContext) []data.Source {

	sources := make([]data.Source, 0)
	cacheManager := data.SourceCacheManager{RootPath: context.RootResource.Dir}

	sources = append(sources, data.LocalSourcesFromEnv()...)
	sources = append(sources, &data.LocalSource{Dir: context.RootResource.Dir})

	nodes := context.ListResourceNodes()

	sourceSet := make(map[string]bool)
	id, _ := context.RootResource.Origin.GetPath(&cacheManager)
	sourceSet[id] = true

	for _, node := range nodes {
		for _, source := range node.Source {
			id, _ := source.GetPath(&cacheManager)
			if _, hasKey := sourceSet[id]; !hasKey {
				sources = append(sources, source)
				sourceSet[id] = true
			}
		}

	}

	return sources
}

type GitRepo struct {
	FullName    string `json:"full_name"`
	Description string
}
type GitRepos struct {
	Items []GitRepo
}

func SearchSource() {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.github.com/search/repositories?q=topic:flekszible&sort=stars", nil)
	req.Header.Add("Accept", "application/vnd.github.mercy-preview+json")
	if err != nil {
		panic(err)
	}
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	if res.StatusCode != 200 {
		panic(errors.New("Github API call was unsuccessfull: " + strconv.Itoa(res.StatusCode)))
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	results := GitRepos{}
	err = json.Unmarshal(body, &results)
	if err != nil {
		panic(err)
	}
	table := termtables.CreateTable()
	table.AddHeaders("name", "description")
	for _, repo := range results.Items {
		table.AddRow("github.com/"+repo.FullName, repo.Description)
	}
	fmt.Println("Available flekszible repositories:")
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println("Add flekszible topic to your repository to show your repository here.")
	fmt.Println()

}
func AddSource(context *processor.RenderContext, inputDir string, source string) error {
	var conf data.Configuration
	conf, configFile, err := data.ReadConfiguration(inputDir)
	if err != nil {
		return errors.Wrap(err, "Can't read existing conf from dir "+inputDir)
	}
	if configFile == "" {
		configFile = path.Join(inputDir, "Flekszible")
	}
	conf.Source = append(conf.Source, data.ConfigSource{Url: source})
	out, err := yaml.Marshal(conf)
	if err != nil {
		return errors.Wrap(err, "Can't write marshall config to yaml "+configFile)
	}
	err = ioutil.WriteFile(configFile, out, 0755)
	if err != nil {
		return errors.Wrap(err, "Can't write out file "+configFile)
	}
	return nil
}

func AddApp(context *processor.RenderContext, inputDir string, app string) {
	var conf data.Configuration
	conf, configFile, err := data.ReadConfiguration(inputDir)
	if err != nil {
		panic(err)
	}
	if configFile == "" {
		configFile = path.Join(inputDir, "Flekszible")
	}
	conf.Import = append(conf.Import, data.ImportConfiguration{Path: app})
	out, err := yaml.Marshal(conf)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(configFile, out, 0755)
	if err != nil {
		panic(err)
	}
}

func Cleanup(context *processor.RenderContext, all bool) error {
	err := context.Init()
	if err != nil {
		return err
	}

	AddInternalTransformations(context, false)
	cleanup := processor.CreateCleanup(context.OutputDir, all)
	context.RootResource.ProcessorRepository.Append(cleanup)
	err = context.Render()
	if err != nil {
		return err
	}
	return nil
}

func ListSources(context *processor.RenderContext) {
	err := context.Init()
	if err != nil {
		panic(err)
	}

	cacheManager := data.SourceCacheManager{RootPath: context.RootResource.Dir}

	table := termtables.CreateTable()
	table.AddHeaders("source", "location", "path")

	for _, source := range listUniqSources(context) {
		typ, value := source.ToString()
		path, _ := source.GetPath(&cacheManager)
		table.AddRow(typ, value, path)
	}
	fmt.Println("Detected sources:")
	fmt.Println(table.Render())
}

func SearchComponent(context *processor.RenderContext) {
	err := context.Init()
	if err != nil {
		panic(err)
	}

	table := termtables.CreateTable()
	table.AddHeaders("path", "description")
	cacheManager := data.SourceCacheManager{RootPath: context.RootResource.Dir}
	for _, source := range listUniqSources(context) {
		findApps(source, &cacheManager, table)

	}
	fmt.Println(table.Render())
}

func findApps(source data.Source, manager *data.SourceCacheManager, table *termtables.Table) {

	dir, err := source.GetPath(manager)
	if dir == "" {
		return
	}
	if err != nil {
		logrus.Error("Can't find real path of the source")
	}
	err = filepath.Walk(dir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == ".cache" {
			return filepath.SkipDir
		}
		if path.Base(filePath) == "flekszible.yaml" {
			relpath, err := filepath.Rel(dir, filepath.Dir(filePath))
			if relpath == "." {
				return nil
			}
			if err != nil {
				logrus.Error("Can't find relative path of" + filePath + " " + err.Error())
			}
			fleksz := make(map[string]interface{})
			bytes, err := ioutil.ReadFile(filePath)
			if err != nil {
				logrus.Error("Can't read flekszible.yaml from " + filePath + " " + err.Error())
			}
			name := ""
			err = yaml.Unmarshal(bytes, &fleksz)
			if err != nil {
				logrus.Error("Can't parse flekszible.yaml from " + filePath + " " + err.Error())
			}
			if declaredName, found := fleksz["description"]; found {
				name = declaredName.(string)
				table.AddRow(relpath, name)
			}
		}

		return nil
	})

}

func ListApp(context *processor.RenderContext) {
	err := context.Init()
	if err != nil {
		panic(err)
	}

	table := termtables.CreateTable()
	table.AddHeaders("dir")

	nodes := context.ListResourceNodes()
	for _, node := range nodes {
		table.AddRow(node.Dir)
	}
	fmt.Println("Detected components (dirs):")
	fmt.Println(table.Render())
}

func Run(context *processor.RenderContext, minikube bool, imports []string, transformations []string) error {
	err := context.Init()
	if err != nil {
		return err
	}
	err = context.AddAdHocTransformations(transformations)
	if err != nil {
		return err
	}
	AddInternalTransformations(context, minikube)
	return context.Render()
}


func AddInternalTransformations(context *processor.RenderContext, minikube bool) {
	if len(context.ImageOverride) > 0 {
		context.RootResource.ProcessorRepository.Append(&processor.Image{
			Image: context.ImageOverride,
		})
	}
	if context.Namespace != "<none>" {
		if len(context.Namespace) > 0 {
			context.RootResource.ProcessorRepository.Append(&processor.Namespace{
				Namespace: context.Namespace,
				Force:     true,
			})
		}
	} else {
		context.Namespace = ""
	}
	if minikube {
		context.RootResource.ProcessorRepository.Append(&processor.DaemonToStatefulSet{})
		context.RootResource.ProcessorRepository.Append(&processor.PublishService{})
	}
	if context.Mode == "k8s" {
		context.RootResource.ProcessorRepository.Append(&processor.K8sWriter{})
	}
}

type GoGetterDownloader struct {
}

func (GoGetterDownloader) Download(url string, destinationDir string, rootPath string) error {
	if os.Getenv("FLEKSZIBLE_OFFLINE") == "true" {
		return nil
	}
	setPwd := func(client *getter.Client) error { client.Pwd = rootPath; return nil }
	return getter.Get(destinationDir, url, setPwd)
}

func init() {
	data.DownloaderPlugin = GoGetterDownloader{}
}
