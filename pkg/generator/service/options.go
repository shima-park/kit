package service

import (
	"io"
	"strings"

	"ezrpro.com/micro/kit/pkg/generator"
	"ezrpro.com/micro/kit/pkg/utils"
)

const (
	ServiceTemplate     Template = "service"
	BaseServiceTemplate Template = "base_service"
	NoopServiceTemplate Template = "noop_service"
	OptionsTemplate     Template = "options"
)

var TemplateMap = map[Template]string{
	ServiceTemplate:     DefaultServiceTemplate,
	BaseServiceTemplate: DefaultBaseServiceTemplate,
	NoopServiceTemplate: DefaultNoopServiceTemplate,
	OptionsTemplate:     DefaultOptionsTemplate,
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
	readWriterMap map[Template]readWriter

	serviceName   string
	methods       []string
	serviceSuffix string
	reqAndResps   []generator.ReqAndResp
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

func WithServiceName(serviceName string) Option {
	return func(o *Options) {
		o.serviceName = serviceName
	}
}

func WithMethods(methods []string) Option {
	return func(o *Options) {
		if len(methods) == 0 {
			return
		}
		o.methods = methods
	}
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

func WithServiceSuffix(serviceSuffix string) Option {
	return func(o *Options) {
		o.serviceSuffix = serviceSuffix
	}
}

func WithRequestAndResponses(reqAndResps []generator.ReqAndResp) Option {
	return func(o *Options) {
		o.reqAndResps = reqAndResps
	}
}
