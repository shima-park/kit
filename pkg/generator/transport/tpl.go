package transport

var defaultTemplate = `
{{$pkg := .PackageName}}
{{$ifaceName := .InterfaceName}}
package addtransport

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc"

	stdopentracing "github.com/opentracing/opentracing-go"
	stdzipkin "github.com/openzipkin/zipkin-go"
	"github.com/sony/gobreaker"
	oldcontext "golang.org/x/net/context"
	"golang.org/x/time/rate"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/tracing/opentracing"
	"github.com/go-kit/kit/tracing/zipkin"
	grpctransport "github.com/go-kit/kit/transport/grpc"

        {{.PackageName}}service "{{.ServiceImportPath}}"
        {{.PackageName}}endpoint "{{.EndpointImportPath}}"
        pb "{{.ProtobufImportPath}}"
)

type grpcServer struct {
{{range $k, $method := .InterfaceMethods}}
	{{ToLowerFirstCamelCase $method.Name}}    grpctransport.Handler
{{end}}
}

// NewGRPCServer makes a set of endpoints available as a gRPC AddServer.
func NewGRPCServer(endpoints addendpoint.Set, otTracer stdopentracing.Tracer, zipkinTracer *stdzipkin.Tracer, logger log.Logger) pb.AddServer {
	// Zipkin GRPC Server Trace can either be instantiated per gRPC method with a
	// provided operation name or a global tracing service can be instantiated
	// without an operation name and fed to each Go kit gRPC server as a
	// ServerOption.
	// In the latter case, the operation name will be the endpoint's grpc method
	// path if used in combination with the Go kit gRPC Interceptor.
	//
	// In this example, we demonstrate a global Zipkin tracing service with
	// Go kit gRPC Interceptor.
	zipkinServer := zipkin.GRPCServerTrace(zipkinTracer)

	options := []grpctransport.ServerOption{
		grpctransport.ServerErrorLogger(logger),
		zipkinServer,
	}

	return &grpcServer{
{{range $k, $method := .InterfaceMethods}}
{{ToLowerFirstCamelCase $method.Name}}: grpctransport.NewServer(
			endpoints.{{$method.Name}}Endpoint,
			decodeGRPC{{$method.Name}}Request,
			encodeGRPC{{$method.Name}}Response,
			append(options, grpctransport.ServerBefore(opentracing.GRPCToContext(otTracer, "{{$method.Name}}", logger)))...,
		),
{{end}}
	}
}

{{range $methodIndex, $method := .InterfaceMethods}}
func (s *grpcServer) {{$method.Name}}(ctx context.Context, req *pb.{{$method.Name}}Request) (resp *pb.{{$method.Name}}Response, err error) {
	_, rep, err := s.{{ToLowerFirstCamelCase $method.Name}}.ServeGRPC({{JoinFieldKeysByComma $method.Params}})
	if err != nil {
		return nil, err
	}
	return rep.(*pb.{{$method.Name}}Response), nil
}
{{end}}

// NewGRPCClient returns an AddService backed by a gRPC server at the other end
// of the conn. The caller is responsible for constructing the conn, and
// eventually closing the underlying transport. We bake-in certain middlewares,
// implementing the client library pattern.
func NewGRPCClient(conn *grpc.ClientConn, otTracer stdopentracing.Tracer, zipkinTracer *stdzipkin.Tracer, logger log.Logger) {{.PackageName}}service.{{ToCamelCase .PackageName}} {
	// We construct a single ratelimiter middleware, to limit the total outgoing
	// QPS from this client to all methods on the remote instance. We also
	// construct per-endpoint circuitbreaker middlewares to demonstrate how
	// that's done, although they could easily be combined into a single breaker
	// for the entire remote instance, too.
	limiter := ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Every(time.Second), 100))

	// Zipkin GRPC Client Trace can either be instantiated per gRPC method with a
	// provided operation name or a global tracing client can be instantiated
	// without an operation name and fed to each Go kit client as ClientOption.
	// In the latter case, the operation name will be the endpoint's grpc method
	// path.
	//
	// In this example, we demonstrace a global tracing client.
	zipkinClient := zipkin.GRPCClientTrace(zipkinTracer)

	// global client middlewares
	options := []grpctransport.ClientOption{
		zipkinClient,
	}

	// Each individual endpoint is an grpc/transport.Client (which implements
	// endpoint.Endpoint) that gets wrapped with various middlewares. If you
	// made your own client library, you'd do this work there, so your server
	// could rely on a consistent set of client behavior.
{{range $k, $method := .InterfaceMethods}}
	var {{ToLowerFirstCamelCase $method.Name}}Endpoint endpoint.Endpoint
	{
		{{ToLowerFirstCamelCase $method.Name}}Endpoint = grpctransport.NewClient(
			conn,
			"{{$pkg}}.{{$ifaceName}}",
			"{{$method.Name}}",
			encodeGRPC{{$method.Name}}Request,
			decodeGRPC{{$method.Name}}Response,
			pb.{{$method.Name}}Response{},
			append(options, grpctransport.ClientBefore(opentracing.ContextToGRPC(otTracer, logger)))...,
		).Endpoint()
		{{ToLowerFirstCamelCase $method.Name}}Endpoint = opentracing.TraceClient(otTracer, "{{$method.Name}}")({{ToLowerFirstCamelCase $method.Name}}Endpoint)
		{{ToLowerFirstCamelCase $method.Name}}Endpoint = limiter({{ToLowerFirstCamelCase $method.Name}}Endpoint)
		{{ToLowerFirstCamelCase $method.Name}}Endpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "{{$method.Name}}",
			Timeout: 30 * time.Second,
		}))({{ToLowerFirstCamelCase $method.Name}}Endpoint)
	}
{{end}}

	// Returning the endpoint.Set as a service.Service relies on the
	// endpoint.Set implementing the Service methods. That's just a simple bit
	// of glue code.
	return addendpoint.Set{
{{range $methodIndex, $method := .InterfaceMethods}}
		{{$method.Name}}Endpoint:    {{ToLowerFirstCamelCase $method.Name}}Endpoint,
{{end}}
	}
}

{{range .RequestsAndResponses}}
{{if .Request}}
// decodeGRPC{{.Request.Name}} is a transport/grpc.DecodeRequestFunc that converts a
// gRPC sum request to a user-domain sum request. Primarily useful in a server.
func decodeGRPC{{.Request.Name}}(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.{{.Request.Name}})
	return &addendpoint.{{.Request.Name}}{
            {{range $fk, $fv := .Request.Fields}}{{$fv.Name}}: {{$fv.Type}}(req.{{$fv.Name}}),
            {{end}}
        }, nil
}
{{end}}

{{if .Response}}
// decodeGRPC{{.Response.Name}} is a transport/grpc.DecodeResponseFunc that converts a
// gRPC sum response to a user-domain sum response. Primarily useful in a client.
func decodeGRPC{{.Response.Name}}(_ context.Context, grpcResponse interface{}) (interface{}, error) {
	resp := grpcResponse.(*pb.{{.Response.Name}})
	return &addendpoint.{{.Response.Name}}{
            V:&{{$pkg}}service.{{.Response.Name}}{
            {{range $fk, $fv := .Response.Fields}}{{$fv.Name}}: {{$fv.Type}}(resp.{{$fv.Name}}),
            {{end}}
        },
        }, nil
}
{{end}}
{{end}}


{{range .PBCST.RequestsAndResponses}}
{{if .Request}}
// encodeGRPC{{.Request.Name}} is a transport/grpc.EncodeRequestFunc that converts a
// user-domain sum request to a gRPC sum request. Primarily useful in a client.
func encodeGRPC{{.Request.Name}}(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(*{{$pkg}}service.{{.Request.Name}})
	return &pb.{{.Request.Name}}{
            {{range $fk, $fv := .Request.Fields}}{{$fv.Name}}: {{$fv.Type}}(req.{{$fv.Name}}),
            {{end}}
        }, nil
}
{{end}}

{{if .Response}}
// encodeGRPC{{.Response.Name}} is a transport/grpc.EncodeResponseFunc that converts a
// user-domain sum response to a gRPC sum response. Primarily useful in a server.
func encodeGRPC{{.Response.Name}}(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(*{{$pkg}}endpoint.{{.Response.Name}})
	return &pb.{{.Response.Name}}{
            {{range $fk, $fv := .Response.Fields}}{{$fv.Name}}: {{$fv.Type}}(resp.V.{{$fv.Name}}),
            {{end}}
        }, nil
}
{{end}}

{{end}}

// These annoying helper functions are required to translate Go error types to
// and from strings, which is the type we use in our IDLs to represent errors.
// There is special casing to treat empty strings as nil errors.

func str2err(s string) error {
	if s == "" {
		return nil
	}
	return errors.New(s)
}

func err2str(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
`
