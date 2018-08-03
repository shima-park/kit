package service

var DefaultServiceTemplate = `
package {{.PackageName}}

import (
        "context"
	"errors"

	"github.com/go-kit/kit/endpoint"
)

// {{.ServiceName}} describes the service.
type {{.ServiceName}} interface {
{{if not .InterfaceMethods}}
    // Add your methods here
    // e.x: Foo(ctx context.Context, *FooRequest)(*FooResponse, err error)
{{else}}
    {{range .InterfaceMethods}}
        {{.}}(ctx context.Context,req *{{.}}Request)(resp *{{.}}Response, err error)
    {{end}}
{{end}}
}

type Middleware func({{.ServiceName}}) {{.ServiceName}}

// New returns a basic Service with all of the expected middlewares wired in.
func New(opts ...Option) {{.ServiceName}} {
	var options Options
	for _, opt := range opts {
		opt(&options)
	}

	if options.service == nil {
		options.service = noopService{}
	}

	for _, middleware := range options.middlewares {
		options.service = middleware(options.service)
	}
	return newBasicService(options)
}

{{if .InterfaceMethods}}
    {{range .InterfaceMethods}}
        type {{.}}Request struct{

        }

        type {{.}}Response struct{
            Code int
            Message string
        }
    {{end}}
{{end}}

var (
{{if .InterfaceMethods}}
    {{range .InterfaceMethods}}
	_ endpoint.Failer = {{.}}Response{}
    {{end}}
{{end}}
)

{{if .InterfaceMethods}}
    {{range .InterfaceMethods}}
// Failed implements endpoint.Failer.
func (r {{.}}Response) Failed() error {
	if r.Code != 0 {
		return errors.New(r.Message)
	}
	return nil
}
    {{end}}
{{end}}
`

var DefaultOptionsTemplate = `
package {{.PackageName}}

type Options struct {
	middlewares []Middleware
	service     {{.ServiceName}}
}

type Option func(*Options)

func WithMiddleware(middlewares ...Middleware) Option {
	return func(o *Options) {
		o.middlewares = append(o.middlewares, middlewares...)
	}
}

func WithService(service {{.ServiceName}}) Option {
	return func(o *Options) {
		o.service = service
	}
}
`

var DefaultBaseServiceTemplate = `
{{$serviceName := .ServiceName}}
package {{.PackageName}}

import "context"

// NewBasicService returns a naïve, stateless implementation of {{.ServiceName}}.
func newBasicService(opts Options) {{.ServiceName}} {
	return basicService{opts: opts}
}

type basicService struct {
	opts Options
}

{{if .InterfaceMethods}}
    {{range .InterfaceMethods}}
// {{.}} implements {{$serviceName}}.
func (s basicService) {{.}}(ctx context.Context, req *{{.}}Request) (resp *{{.}}Response, err error) {
	return s.opts.service.{{.}}(ctx, req)
}
    {{end}}
{{end}}
`

var DefaultNoopServiceTemplate = `
{{$serviceName := .ServiceName}}
package {{.PackageName}}

import "context"

type noopService struct{}

{{if .InterfaceMethods}}
    {{range .InterfaceMethods}}
// {{.}} implements {{$serviceName}}.
func (n noopService) {{.}}(ctx context.Context, req *{{.}}Request) (resp *{{.}}Response, err error) {
	return &{{.}}Response{}, nil
}
    {{end}}
{{end}}
`
