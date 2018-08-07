package cst

import "strings"

type Options struct {
	fieldNameFilter FieldNameFilter
	packageName     string
}

type Option func(*Options)

type FieldNameFilter func(fieldName string) bool

func DefaultFieldNameFilter(fieldName string) bool {
	switch {
	case strings.HasPrefix(fieldName, "XXX_"):
		return true
	}
	return false
}

func WithFieldNameFilter(f FieldNameFilter) Option {
	return func(o *Options) {
		o.fieldNameFilter = f
	}
}

func WithPackageName(pkg string) Option {
	return func(o *Options) {
		o.packageName = pkg
	}
}
