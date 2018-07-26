package service

import (
	"io/ioutil"
	"strings"

	"text/template"

	gen "ezrpro.com/micro/kit/pkg/generator"
	"ezrpro.com/micro/kit/pkg/utils"
)

type ServiceGenerator struct {
	opts Options
}

func NewServiceGenerator(opts ...Option) gen.Generator {
	var options Options
	for _, opt := range opts {
		opt(&options)
	}

	if options.writer == nil {
		options.writer = gen.DefaultWriter
	}

	if options.tpl == nil {
		options.tpl = strings.NewReader(defaultTemplate)
	}

	return &ServiceGenerator{
		opts: options,
	}
}

func (g *ServiceGenerator) Generate() error {
	tplBody, err := ioutil.ReadAll(g.opts.tpl)
	if err != nil {
		return err
	}

	t := template.New("service").Funcs(map[string]interface{}{
		"ToCamelCase":           utils.ToCamelCase,
		"ToLowerFirstCamelCase": utils.ToLowerFirstCamelCase,
		"ToLowerSnakeCase":      utils.ToLowerSnakeCase,
		"ToUpperFirst":          utils.ToUpperFirst,
		"ToLower":               strings.ToLower,
	})
	t, err = t.Parse(string(tplBody))
	if err != nil {
		return err
	}

	var data = map[string]interface{}{
		"ServiceName":      g.opts.serviceName,
		"InterfaceMethods": g.opts.methods,
	}

	return t.Execute(g.opts.writer, data)
}
