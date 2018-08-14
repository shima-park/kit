package service

import (
	"io/ioutil"
	"strings"

	gen "ezrpro.com/micro/kit/pkg/generator"
	"ezrpro.com/micro/kit/pkg/utils"
	"github.com/alecthomas/template"
)

type ServiceGenerator struct {
	opts Options
}

func NewServiceGenerator(opts ...Option) gen.Generator {
	options := newOptions(opts...)

	if !strings.HasSuffix(options.serviceName, options.serviceSuffix) {
		options.serviceName = utils.ToCamelCase(options.serviceName + options.serviceSuffix)
	}

	for i, method := range options.methods {
		options.methods[i] = utils.ToCamelCase(method)
	}

	return &ServiceGenerator{
		opts: options,
	}
}

func (g *ServiceGenerator) Generate() error {
	for tplName, readWriter := range g.opts.readWriterMap {

		tplBody, err := ioutil.ReadAll(readWriter.template)
		if err != nil {
			return err
		}

		t := template.New(string(tplName))
		t, err = t.Parse(string(tplBody))
		if err != nil {
			return err
		}

		packageName := strings.ToLower(g.opts.serviceName)

		var data = map[string]interface{}{
			"PackageName":         packageName,
			"ServiceName":         g.opts.serviceName,
			"InterfaceMethods":    g.opts.methods,
			"RequestAndResponses": g.opts.reqAndResps,
		}

		err = t.Execute(readWriter.writer, data)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetBaseServiceName(packageName, serviceSuffix string) string {
	packageName = strings.ToLower(packageName)
	serviceSuffix = strings.ToLower(serviceSuffix)
	return strings.TrimSuffix(packageName, serviceSuffix)
}
