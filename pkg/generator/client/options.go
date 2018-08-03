package client

import (
	"io"
	"strings"

	"ezrpro.com/micro/kit/pkg/utils"
)

const (
	ClientTemplate   Template = "client"
	OptionsTempalte  Template = "options"
	ExportedTemplate Template = "exported"
)

var TemplateMap = map[Template]string{
	ClientTemplate:   DefaultClientTemplate,
	OptionsTempalte:  DefaultOptionsTemplate,
	ExportedTemplate: DefaultExportedTemplate,
}

type Template string

func (t Template) String() string {
	return string(t)
}

type readWriter struct {
	writer   io.Writer
	template io.Reader
}

type Options struct {
	readWriterMap     map[Template]readWriter
	baseServiceName   string
	clientPackageName string
	serviceSuffix     string
}

type Option func(*Options)

func newOptions(opts ...Option) Options {
	var options Options
	for _, opt := range opts {
		opt(&options)
	}

	if options.serviceSuffix == "" {
		options.serviceSuffix = utils.GetServiceSuffix()
	}
	return options
}

func WithReadWriter(t Template, tpl io.Reader, w io.Writer) Option {
	return func(o *Options) {
		if o.readWriterMap == nil {
			o.readWriterMap = map[Template]readWriter{}
		}
		if tpl == nil {
			tpl = strings.NewReader(TemplateMap[t])
		}
		o.readWriterMap[t] = readWriter{
			writer:   w,
			template: tpl,
		}
	}
}

func WithBaseServiceName(baseServiceName string) Option {
	return func(o *Options) {
		o.baseServiceName = baseServiceName
	}
}

func WithClientPackageName(clientPackageName string) Option {
	return func(o *Options) {
		o.clientPackageName = clientPackageName
	}
}

func WithServiceSuffix(serviceSuffix string) Option {
	return func(o *Options) {
		o.serviceSuffix = serviceSuffix
	}
}
