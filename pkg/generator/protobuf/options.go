package protobuf

import (
	"io"

	gen "ezrpro.com/micro/kit/pkg/generator"
)

type Options struct {
	serviceNameNormalizer gen.Normalizer
	fieldNameNormalizer   gen.Normalizer
	typeFilter            gen.TypeFilter
	structFilter          gen.StructFilter
	writer                io.Writer
}

type Option func(*Options)

func WithFieldNameNormalizer(normalizer gen.Normalizer) Option {
	return func(o *Options) {
		o.serviceNameNormalizer = normalizer
	}
}

func WithServiceNameNormalizer(normalizer gen.Normalizer) Option {
	return func(o *Options) {
		o.fieldNameNormalizer = normalizer
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
