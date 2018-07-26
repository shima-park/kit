package endpoint

var defaultTemplate = `
{{$pkg := .PackageName}}
{{$ifaceName := .InterfaceName}}
// THIS FILE IS AUTO GENERATED DO NOT EDIT!!
package {{$pkg}}

import (
	"context"
	"time"

	"golang.org/x/time/rate"

	stdopentracing "github.com/opentracing/opentracing-go"
	stdzipkin "github.com/openzipkin/zipkin-go"
	"github.com/sony/gobreaker"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/tracing/opentracing"
	"github.com/go-kit/kit/tracing/zipkin"

	"{{.ServiceImportPath}}"
)

// Set collects all of the endpoints that compose an add service. It's meant to
// be used as a helper struct, to collect all of the endpoints into a single
// parameter.
type Set struct {
{{range $k, $method := .InterfaceMethods}}
	{{$method.Name}}Endpoint    endpoint.Endpoint
{{end}}
}

// New returns a Set that wraps the provided server, and wires in all of the
// expected endpoint middlewares via the various parameters.
func New(svc {{.PackageName}}.{{ToCamelCase .PackageName}}, logger log.Logger, duration metrics.Histogram, otTracer stdopentracing.Tracer, zipkinTracer *stdzipkin.Tracer) Set {
{{range $k, $method := .InterfaceMethods}}
	var {{ToLowerFirstCamelCase $method.Name}} endpoint.Endpoint
	{
		{{ToLowerFirstCamelCase $method.Name}} = Make{{$method.Name}}Endpoint(svc)
		{{ToLowerFirstCamelCase $method.Name}} = ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Every(time.Second), 1))({{ToLowerFirstCamelCase $method.Name}})
		{{ToLowerFirstCamelCase $method.Name}} = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))({{ToLowerFirstCamelCase $method.Name}})
		{{ToLowerFirstCamelCase $method.Name}} = opentracing.TraceServer(otTracer, "{{$method.Name}}")({{ToLowerFirstCamelCase $method.Name}})
		{{ToLowerFirstCamelCase $method.Name}} = zipkin.TraceEndpoint(zipkinTracer, "{{$method.Name}}")({{ToLowerFirstCamelCase $method.Name}})
		{{ToLowerFirstCamelCase $method.Name}} = LoggingMiddleware(log.With(logger, "method", "{{$method.Name}}"))({{ToLowerFirstCamelCase $method.Name}})
		{{ToLowerFirstCamelCase $method.Name}} = InstrumentingMiddleware(duration.With("method", "{{$method.Name}}"))({{ToLowerFirstCamelCase $method.Name}})
	}
{{end}}

	return Set{
{{range $k, $method := .InterfaceMethods}}
	{{$method.Name}}Endpoint:    {{ToLowerFirstCamelCase $method.Name}},
{{end}}
	}
}

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

{{range $methodIndex, $method := .InterfaceMethods}}
// {{$method.Name}} implements the service interface, so Set may be used as a service.
// This is primarily useful in the context of a client library.
func (s Set) {{$method.Name}}(ctx context.Context, req *{{$pkg}}.{{$method.Name}}Request) (resp *{{$pkg}}.{{$method.Name}}Response, err error) {
	temp, err := s.{{$method.Name}}Endpoint({{JoinFieldKeysByComma $method.Params}})
	if err != nil {
		return
	}
	response := temp.(*{{$method.Name}}Response)
	return response.V, response.Err
}
{{end}}

{{range .RequestsAndResponses}}
// Make{{.MethodName}}Endpoint constructs a {{.MethodName}} endpoint wrapping the service.
func Make{{.MethodName}}Endpoint(s {{$pkg}}.{{$ifaceName}}) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (resp interface{}, err error) {
		req := request.(*{{.Request.Name}})
		v, err := s.{{.MethodName}}(ctx, &{{$pkg}}.{{.Request.Name}}{
                    {{range $fk, $fv := .Request.Fields}}{{$fv.Name}}: req.{{$fv.Name}},
                    {{end}}
                })
		return &{{.MethodName}}Response{V: v, Err: err}, nil
	}
}
{{end}}


// compile time assertions for our response types implementing endpoint.Failer.
var (
{{range .Responses}}
	_ endpoint.Failer = {{.Name}}{}
{{end}}
)

{{range .RequestsAndResponses}}

// {{.Request.Name}} collects the request parameters for the Sum method.
type {{.Request.Name}} struct {
{{JoinFieldsByLineBreak .Request.Fields}}
}

// {{.Response.Name}} collects the response values for the Sum method.
type {{.Response.Name}} struct {
	V   *{{$pkg}}.{{.Response.Name}}   ` + "`json:\"v\"`" + `
	Err error ` + "`json:\"-\"`" + ` // should be intercepted by Failed/errorEncoder
}

// Failed implements endpoint.Failer.
func (r {{.Response.Name}}) Failed() error { return r.Err }

{{end}}
`
