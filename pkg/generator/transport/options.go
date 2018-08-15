package transport

import (
	"io"
	"strings"

	"ezrpro.com/micro/kit/pkg/utils"
)

const (
	GRPCTemplate    Template = "grpc"
	HTTPTemplate    Template = "http"
	OptionsTempalte Template = "options"
)

var TemplateMap = map[Template]string{
	GRPCTemplate:    DefaultGRPCTemplate,
	HTTPTemplate:    DefaultHTTPTemplate,
	OptionsTempalte: DefaultOptionsTemplate,
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
	readWriterMap        map[Template]readWriter
	transportPackageName string
	baseServiceName      string
	serviceSuffix        string
	pbGoPath             string
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

func WithTransportPackageName(transportPackageName string) Option {
	return func(o *Options) {
		o.transportPackageName = transportPackageName
	}
}

func WithServiceSuffix(serviceSuffix string) Option {
	return func(o *Options) {
		o.serviceSuffix = serviceSuffix
	}
}

func WithPBGoPath(pbGoPath string) Option {
	return func(o *Options) {
		o.pbGoPath = pbGoPath
	}
}
