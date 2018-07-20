package endpoint

import (
	"io"

	gen "ezrpro.com/micro/kit/pkg/generator"
)

type Options struct {
	writer    io.Writer
	tpl       io.Reader
	tplConfig gen.TemplateConfig
}

type Option func(*Options)

func WithWriter(w io.Writer) Option {
	return func(o *Options) {
		o.writer = w
	}
}

func WithTemplate(r io.Reader) Option {
	return func(o *Options) {
		o.tpl = r
	}
}

func WithTemplateConfig(tc gen.TemplateConfig) Option {
	return func(o *Options) {
		o.tplConfig = tc
	}
}
