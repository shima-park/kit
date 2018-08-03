package endpoint

var DefaultEndpointTemplate = `
{{$servicePackageName := BasePath .ServiceImportPath}}
{{$serviceName := .ServiceName}}
package {{.PackageName}}

import (
	"context"

	"ezrpro.com/micro/spiderconn"
        {{$servicePackageName}} "{{.ServiceImportPath}}"
)

// Set collects all of the endpoints that compose an add service. It's meant to
// be used as a helper struct, to collect all of the endpoints into a single
// parameter.
type Set struct {
{{range $index, $method := .ServiceMethods}}
	{{$method.Name}}Endpoint    spiderconn.EndpointWrapper
{{end}}
}

// New returns a Set that wraps the provided server, and wires in all of the
// expected endpoint middlewares via the various parameters.
func New(opts ...Option) *Set {
	var options Options
	for _, opt := range opts {
		opt(&options)
	}

	if options.service == nil {
		options.service = {{$servicePackageName}}.New(options.serviceOptions...)
	}

{{range $index, $method := .ServiceMethods}}
	var {{ToLowerFirstCamelCase $method.Name}} spiderconn.EndpointWrapper
	{
		{{ToLowerFirstCamelCase $method.Name}} = MakeSumEndpoint(options.service)
		for _, middlewareCreator := range options.middlewareCreators {
			{{ToLowerFirstCamelCase $method.Name}}.Wrapper(middlewareCreator({{ToLowerFirstCamelCase $method.Name}}.Name()))
		}
	}
{{end}}

	return &Set{
{{range $index, $method := .ServiceMethods}}
		{{$method.Name}}Endpoint: {{ToLowerFirstCamelCase $method.Name}},
{{end}}
	}
}

{{range $index, $method := .ServiceMethods}}
// {{$method.Name}} implements the service interface, so Set may be used as a service.
// This is primarily useful in the context of a client library.
func (s Set) {{$method.Name}}(ctx context.Context, req *{{$servicePackageName}}.{{$method.Name}}Request) (resp *{{$servicePackageName}}.{{$method.Name}}Response, err error) {
	temp, err := s.{{$method.Name}}Endpoint.Do(ctx, req)
	if err != nil {
		return
	}
	response := temp.(*{{$servicePackageName}}.{{$method.Name}}Response)
	return response, response.Failed()
}

// Make{{$method.Name}}Endpoint constructs a {{$method.Name}} endpoint wrapping the service.
func Make{{$method.Name}}Endpoint(s {{$servicePackageName}}.{{$serviceName}}) spiderconn.EndpointWrapper {
	return spiderconn.NewWrapper("{{$method.Name}}", func(ctx context.Context, request interface{}) (resp interface{}, err error) {
		req := request.(*{{$servicePackageName}}.{{$method.Name}}Request)
		return s.{{$method.Name}}(ctx, req)
	})
}
{{end}}
`

var DefaultOptionsTemplate = `
{{$servicePackageName := BasePath .ServiceImportPath}}
package {{.PackageName}}

import (
	{{$servicePackageName}} "{{.ServiceImportPath}}"
	"ezrpro.com/micro/spiderconn/middleware"
)

type Options struct {
	middlewareCreators []middleware.Creator
	service            {{$servicePackageName}}.{{.ServiceName}}
	serviceOptions     []{{$servicePackageName}}.Option
}

type Option func(*Options)

func WithMiddlewareCreators(cs ...middleware.Creator) Option {
	return func(o *Options) {
		o.middlewareCreators = append(o.middlewareCreators, cs...)
	}
}

func WithService(service {{$servicePackageName}}.{{.ServiceName}}) Option {
	return func(o *Options) {
		o.service = service
	}
}

func WithServiceOptions(opts ...{{$servicePackageName}}.Option) Option {
	return func(o *Options) {
		o.serviceOptions = append(o.serviceOptions, opts...)
	}
}
`

var DefaultMiddlewareTemplate = `
package {{.PackageName}}

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
)

// InstrumentingMiddleware returns an endpoint middleware that records
// the duration of each invocation to the passed histogram. The middleware adds
// a single field: "success", which is "true" if no error is returned, and
// "false" otherwise.
func InstrumentingMiddleware(duration metrics.Histogram) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {

			defer func(begin time.Time) {
				duration.With("success", fmt.Sprint(err == nil)).Observe(time.Since(begin).Seconds())
			}(time.Now())
			return next(ctx, request)

		}
	}
}

// LoggingMiddleware returns an endpoint middleware that logs the
// duration of each invocation, and the resulting error, if any.
func LoggingMiddleware(logger log.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {

			defer func(begin time.Time) {
				logger.Log("transport_error", err, "took", time.Since(begin))
			}(time.Now())
			return next(ctx, request)

		}
	}
}
`
