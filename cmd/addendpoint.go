// THIS FILE IS AUTO GENERATED DO NOT EDIT!!
package addendpoint

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

	"addservice"
)

// Set collects all of the endpoints that compose an add service. It's meant to
// be used as a helper struct, to collect all of the endpoints into a single
// parameter.
type Set struct {
	SumEndpoint endpoint.Endpoint

	ConcatEndpoint endpoint.Endpoint
}

// New returns a Set that wraps the provided server, and wires in all of the
// expected endpoint middlewares via the various parameters.
func New(svc addservice.Service, logger log.Logger, duration metrics.Histogram, otTracer stdopentracing.Tracer, zipkinTracer *stdzipkin.Tracer) Set {

	var sum endpoint.Endpoint
	{
		sum = MakeSumEndpoint(svc)
		sum = ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Every(time.Second), 1))(sum)
		sum = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(sum)
		sum = opentracing.TraceServer(otTracer, "Sum")(sum)
		sum = zipkin.TraceEndpoint(zipkinTracer, "Sum")(sum)
		sum = LoggingMiddleware(log.With(logger, "method", "Sum"))(sum)
		sum = InstrumentingMiddleware(duration.With("method", "Sum"))(sum)
	}

	var concat endpoint.Endpoint
	{
		concat = MakeSumEndpoint(svc)
		concat = ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Every(time.Second), 1))(concat)
		concat = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(concat)
		concat = opentracing.TraceServer(otTracer, "Concat")(concat)
		concat = zipkin.TraceEndpoint(zipkinTracer, "Concat")(concat)
		concat = LoggingMiddleware(log.With(logger, "method", "Concat"))(concat)
		concat = InstrumentingMiddleware(duration.With("method", "Concat"))(concat)
	}

	return Set{

		SumEndpoint: sum,

		ConcatEndpoint: concat,
	}
}

// Sum implements the service interface, so Set may be used as a service.
// This is primarily useful in the context of a client library.
func (s Set) Sum(_ context.Context, req SumRequest) (*SumResponse, error) {
	resp, err := s.SumEndpoint(_, req)
	if err != nil {
		return 0, err
	}
	response := resp.(SumResponse)
	return response.V, response.Err
}

// Concat implements the service interface, so Set may be used as a service.
// This is primarily useful in the context of a client library.
func (s Set) Concat(_ context.Context, req *ConcatRequest) (ConcatResponse, error) {
	resp, err := s.ConcatEndpoint(_, req)
	if err != nil {
		return 0, err
	}
	response := resp.(ConcatResponse)
	return response.V, response.Err
}

// MakeSumEndpoint constructs a Sum endpoint wrapping the service.
func MakeSumEndpoint(s addservice.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(SumRequest)
		v, err := s.Sum(ctx, req)
		return SumResponse{V: v, Err: err}, nil
	}
}

// MakeConcatEndpoint constructs a Concat endpoint wrapping the service.
func MakeSumEndpoint(s addservice.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(ConcatRequest)
		v, err := s.Concat(ctx, req)
		return ConcatResponse{V: v, Err: err}, nil
	}
}

// compile time assertions for our response types implementing endpoint.Failer.
var (
	_ endpoint.Failer = SumResponse{}

	_ endpoint.Failer = ConcatResponse{}
)

// SumRequest collects the request parameters for the Sum method.
type SumRequest struct {
	a int
	b int
	f *model.Foo2
	c int
}

// SumResponse collects the response values for the Sum method.
type SumResponse struct {
	V   SumResponse `json:"v"`
	Err error       `json:"-"` // should be intercepted by Failed/errorEncoder
}

// Failed implements endpoint.Failer.
func (r SumResponse) Failed() error { return r.Err }

// ConcatRequest collects the request parameters for the Sum method.
type ConcatRequest struct {
	a string
	b string
	F Foo
}

// ConcatResponse collects the response values for the Sum method.
type ConcatResponse struct {
	V   ConcatResponse `json:"v"`
	Err error          `json:"-"` // should be intercepted by Failed/errorEncoder
}

// Failed implements endpoint.Failer.
func (r ConcatResponse) Failed() error { return r.Err }
