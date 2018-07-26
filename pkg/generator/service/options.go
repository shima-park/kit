package service

import (
	"io"
)

type Options struct {
	writer io.Writer
	tpl    io.Reader

	serviceName string
	methods     []string
}

type Option func(*Options)

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
