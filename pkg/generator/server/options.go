package server

import (
	"io"
	"strings"

	"ezrpro.com/micro/kit/pkg/utils"
)

const (
	ServerTemplate  Template = "server"
	OptionsTempalte Template = "options"
)

var TemplateMap = map[Template]string{
	ServerTemplate:  DefaultServerTemplate,
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
	readWriterMap     map[Template]readWriter
	baseServiceName   string
	serverPackageName string
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

func WithServerPackageName(serverPackageName string) Option {
	return func(o *Options) {
		o.serverPackageName = serverPackageName
	}
}

func WithServiceSuffix(serviceSuffix string) Option {
	return func(o *Options) {
		o.serviceSuffix = serviceSuffix
	}
}
