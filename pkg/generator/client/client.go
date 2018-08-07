package client

import (
	"io/ioutil"
	"path/filepath"
	"text/template"

	"ezrpro.com/micro/kit/pkg/cst"
	gen "ezrpro.com/micro/kit/pkg/generator"
	"ezrpro.com/micro/kit/pkg/utils"
)

type ClientGenerator struct {
	cst  cst.ConcreteSyntaxTree
	opts Options
}

func NewClientGenerator(t cst.ConcreteSyntaxTree, opts ...Option) gen.Generator {
	options := newOptions(opts...)

	return &ClientGenerator{
		cst:  t,
		opts: options,
	}
}

func (g *ClientGenerator) Generate() error {
	for tplName, readWriter := range g.opts.readWriterMap {
		tplBody, err := ioutil.ReadAll(readWriter.template)
		if err != nil {
			return err
		}

		t := template.New(string(tplName)).Funcs(map[string]interface{}{
			"BasePath":              filepath.Base,
			"ToLowerFirstCamelCase": utils.ToLowerFirstCamelCase,
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
			"BaseServiceName":     g.opts.baseServiceName,
			"PackageName":         g.opts.clientPackageName,
			"ServiceName":         serviceIface.Name,
			"ServiceMethods":      serviceIface.Methods,
			"ServiceImportPath":   utils.GetServiceImportPath(g.opts.baseServiceName),
			"EndpointImportPath":  utils.GetEndpointImportPath(g.opts.baseServiceName),
			"ProtobufImportPath":  utils.GetProtobufImportPath(g.opts.baseServiceName),
			"TransportImportPath": utils.GetTransportImportPath(g.opts.baseServiceName),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
