package server

import (
	"io/ioutil"
	"path/filepath"
	"text/template"

	"ezrpro.com/micro/kit/pkg/cst"
	gen "ezrpro.com/micro/kit/pkg/generator"
	"ezrpro.com/micro/kit/pkg/utils"
)

type ServerGenerator struct {
	cst  cst.ConcreteSyntaxTree
	opts Options
}

func NewServerGenerator(t cst.ConcreteSyntaxTree, opts ...Option) gen.Generator {
	options := newOptions(opts...)

	return &ServerGenerator{
		cst:  t,
		opts: options,
	}
}

func (g *ServerGenerator) Generate() error {
	for tplName, readWriter := range g.opts.readWriterMap {
		tplBody, err := ioutil.ReadAll(readWriter.template)
		if err != nil {
			return err
		}

		t := template.New(string(tplName)).Funcs(map[string]interface{}{
			"BasePath":    filepath.Base,
			"ToCamelCase": utils.ToCamelCase,
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
			"PackageName":         g.opts.serverPackageName,
			"ServiceName":         serviceIface.Name,
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
