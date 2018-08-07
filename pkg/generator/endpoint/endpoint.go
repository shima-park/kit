package endpoint

import (
	"io/ioutil"
	"path/filepath"
	"text/template"

	"ezrpro.com/micro/kit/pkg/cst"
	gen "ezrpro.com/micro/kit/pkg/generator"
	"ezrpro.com/micro/kit/pkg/generator/service"
	"ezrpro.com/micro/kit/pkg/utils"
)

const (
	EndpointSuffix = "Endpoint"
)

type EndpointGenerator struct {
	cst  cst.ConcreteSyntaxTree
	opts Options
}

func NewEndpointGenerator(t cst.ConcreteSyntaxTree, opts ...Option) gen.Generator {
	options := newOptions(opts...)

	if options.baseServiceName == "" {
		options.baseServiceName = service.GetBaseServiceName(
			t.PackageName(),
			options.serviceSuffix,
		)
	}

	return &EndpointGenerator{
		cst:  t,
		opts: options,
	}
}

func (g *EndpointGenerator) Generate() error {
	for tplName, readWriter := range g.opts.readWriterMap {
		tplBody, err := ioutil.ReadAll(readWriter.template)
		if err != nil {
			return err
		}

		t := template.New(string(tplName)).Funcs(map[string]interface{}{
			"ToLowerFirstCamelCase": utils.ToLowerFirstCamelCase,
			"BasePath":              filepath.Base,
		})
		t, err = t.Parse(string(tplBody))
		if err != nil {
			return err
		}

		serviceIface, err := gen.FilterInterface(g.cst.Interfaces(), g.opts.serviceSuffix)
		if err != nil {
			return err
		}

		err = t.Execute(readWriter.writer, map[string]interface{}{
			"PackageName":       g.opts.endpointPackageName,
			"ServiceName":       serviceIface.Name,
			"ServiceMethods":    serviceIface.Methods,
			"ServiceImportPath": utils.GetServiceImportPath(g.opts.baseServiceName),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
