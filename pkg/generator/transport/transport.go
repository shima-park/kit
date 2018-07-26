package transport

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"

	"ezrpro.com/micro/kit/pkg/cst"
	gen "ezrpro.com/micro/kit/pkg/generator"
	"ezrpro.com/micro/kit/pkg/utils"
)

type TransportGenerator struct {
	cst  cst.ConcreteSyntaxTree
	opts Options
}

func NewTransportGenerator(t cst.ConcreteSyntaxTree, opts ...Option) gen.Generator {
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

	return &TransportGenerator{
		cst:  t,
		opts: options,
	}
}

func (g *TransportGenerator) Generate() error {
	tplBody, err := ioutil.ReadAll(g.opts.tpl)
	if err != nil {
		return err
	}

	pbGoPath := utils.GetProtobufFilePath(g.cst.PackageName())
	pbGoFile := filepath.Join(pbGoPath, g.cst.PackageName()+".pb.go")
	pbCST, err := cst.New(pbGoFile)
	if err != nil {
		return err
	}

	t := template.New("transport").Funcs(g.opts.tplConfig.Funcs())
	t, err = t.Parse(string(tplBody))
	if err != nil {
		return err
	}

	data := g.opts.tplConfig.Data()
	data["PBCST"] = gen.NewTemplateConfig(pbCST).Data()

	return t.Execute(g.opts.writer, data)
}
