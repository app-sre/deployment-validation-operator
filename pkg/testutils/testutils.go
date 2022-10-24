package testutils

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path"
	"runtime"
	"text/template"

	"github.com/ghodss/yaml"
	"github.com/mcuadros/go-defaults"
	appsv1 "k8s.io/api/apps/v1"
)

type TemplateArgs struct {
	Replicas         int  `default:"3"`
	ResourceLimits   bool `default:"true"`
	ResourceRequests bool `default:"true"`
}

func NewTemplateArgs() *TemplateArgs {
	args := new(TemplateArgs)
	defaults.SetDefaults(args)
	return args
}

func CreateDeploymentFromTemplate(args *TemplateArgs) (appsv1.Deployment, error) {
	var deployment appsv1.Deployment

	yamlManifest, err := createYamlManifest("Deployment", args)
	if err != nil {
		return deployment, err
	}

	// deserialise from YAML by using the json struct tags that are defined in the k8s API object structs
	err = yaml.Unmarshal(yamlManifest, &deployment)
	if err != nil {
		return deployment, err
	}

	return deployment, nil
}

func createYamlManifest(objectType string, args *TemplateArgs) ([]byte, error) {
	tpl, err := template.ParseFiles(templatePath(objectType))
	if err != nil {
		return nil, err
	}

	templateContent := &bytes.Buffer{}
	if err := tpl.Execute(templateContent, args); err != nil {
		return nil, err
	}

	manifest, err := ioutil.ReadAll(templateContent)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}

func templatePath(objectName string) string {
	_, thisFile, _, _ := runtime.Caller(0)
	return path.Join(path.Dir(thisFile), fmt.Sprintf("templates/%s.yaml.tpl", objectName))
}
