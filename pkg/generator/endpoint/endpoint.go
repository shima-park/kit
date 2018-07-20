package endpoint

import (
	"html/template"
	"io/ioutil"
	"strings"

	"ezrpro.com/micro/kit/pkg/cst"
	gen "ezrpro.com/micro/kit/pkg/generator"
)

type EndpointGenerator struct {
	cst           cst.ConcreteSyntaxTree
	template      *template.Template
	opts          Options
	referenceType map[string]struct{} // key: [struct.Name or type.Name] val: struct{}{}
}

func NewEndpointGenerator(t cst.ConcreteSyntaxTree, opts ...Option) gen.Generator {
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

	if options.tplConfig == nil {
		options.tplConfig = gen.NoopTemplateConifg
	}

	return &EndpointGenerator{
		cst:           t,
		opts:          options,
		referenceType: map[string]struct{}{},
	}
}

func (g *EndpointGenerator) Generate() error {
	tplBody, err := ioutil.ReadAll(g.opts.tpl)
	if err != nil {
		return err
	}

	t := template.New("endpoint").Funcs(g.opts.tplConfig.Funcs())
	t, err = t.Parse(string(tplBody))
	if err != nil {
		return err
	}

	return t.Execute(g.opts.writer, g.opts.tplConfig.Data())
}
