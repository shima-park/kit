package generator

import "io"

type GeneratorConfig struct {
	Options GeneratorConfigOptions
}

func NewGeneratorConfig(opts ...GeneratorConfigOption) *GeneratorConfig {
	var options GeneratorConfigOptions
	for _, opt := range opts {
		opt(&options)
	}

	if options.Generator == nil {
		options.Generator = NoopGenerator
	}

	if options.Writer == nil {
		options.Writer = DefaultWriter
	}

	return &GeneratorConfig{
		Options: options,
	}
}

type GeneratorConfigOptions struct {
	Writer    io.Writer
	Generator Generator
}

type GeneratorConfigOption func(*GeneratorConfigOptions)

func WithGenerator(gen Generator) GeneratorConfigOption {
	return func(o *GeneratorConfigOptions) {
		o.Generator = gen
	}
}

func WithWriter(w Writer) GeneratorConfigOption {
	return func(o *GeneratorConfigOptions) {
		o.Writer = w
	}
}
