package implement

import (
	"io"
	"os"

	"ezrpro.com/micro/kit/pkg/utils"
)

type Options struct {
	sourceDirctory string
	packageName    string
	receiver       string
	iface          string
	writer         io.Writer
}

type Option func(*Options)

func newOptions(opts ...Option) Options {
	var options Options
	for _, opt := range opts {
		opt(&options)
	}

	if options.sourceDirctory == "" {
		options.sourceDirctory = utils.GetPWD()
	}

	if options.writer == nil {
		options.writer = os.Stdout
	}

	return options
}

func WithSourceDirctory(srcDir string) Option {
	return func(o *Options) {
		o.sourceDirctory = srcDir
	}
}

func WithWriter(w io.Writer) Option {
	return func(o *Options) {
		o.writer = w
	}
}

func WithPackageName(pkg string) Option {
	return func(o *Options) {
		o.packageName = pkg
	}
}
