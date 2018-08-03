package protobuf

import (
	"io"

	gen "ezrpro.com/micro/kit/pkg/generator"
	"ezrpro.com/micro/kit/pkg/utils"
)

type Options struct {
	serviceNameNormalizer gen.Normalizer
	fieldNameNormalizer   gen.Normalizer
	typeFilter            gen.TypeFilter
	structFilter          gen.StructFilter
	writer                io.Writer
	serviceSuffix         string
}

type Option func(*Options)

func newOptions(opts ...Option) Options {
	var options Options
	for _, opt := range opts {
		opt(&options)
	}

	if options.serviceNameNormalizer == nil {
		options.serviceNameNormalizer = gen.NoopNormalizer
	}

	if options.fieldNameNormalizer == nil {
		options.fieldNameNormalizer = gen.NoopNormalizer
	}

	if options.typeFilter == nil {
		options.typeFilter = gen.DefaultTypeFilter
	}

	if options.structFilter == nil {
		options.structFilter = gen.DefaultStructFilter
	}

	if options.writer == nil {
		options.writer = gen.DefaultWriter
	}

	if options.serviceSuffix == "" {
		options.serviceSuffix = utils.GetServiceSuffix()
	}
	return options
}

func WithFieldNameNormalizer(normalizer gen.Normalizer) Option {
	return func(o *Options) {
		o.fieldNameNormalizer = normalizer
	}
}

func WithServiceNameNormalizer(normalizer gen.Normalizer) Option {
	return func(o *Options) {
		o.serviceNameNormalizer = normalizer
	}
}

func WithTypeFilter(typeFilter gen.TypeFilter) Option {
	return func(o *Options) {
		o.typeFilter = typeFilter
	}
}

func WithStructFilter(structFilter gen.StructFilter) Option {
	return func(o *Options) {
		o.structFilter = structFilter
	}
}

func WithWriter(w io.Writer) Option {
	return func(o *Options) {
		o.writer = w
	}
}

func WithServiceSuffix(serviceSuffix string) Option {
	return func(o *Options) {
		o.serviceSuffix = serviceSuffix
	}
}
