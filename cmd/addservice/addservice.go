package addservice

import (
	"context"
	"errors"
	"fmt"

	"ezrpro.com/micro/spiderconn/pkg/addservice/model"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
)

var (
	a, b int         = 1, 2
	c    int         = 3
	d    string      = "string"
	f    float64     = 2.3
	g    *int        = &a
	k                = 4
	kk               = "string"
	ff               = 3.3
	ee               = errors.New("xxx")
	h                = &Foo{}
	j                = &Foo{}
	l    *Foo        = &Foo{}
	m    Foo         = Foo{}
	n    model.Foo2  = model.Foo2{}
	vv   *model.Foo2 = &model.Foo2{}
	hh   string
)

const (
	q      = 1
	w      = "w"
	e      = 3.2
	qq int = 1
)

// Service describes a service that adds things together.
type Service interface {
	Sum(ctx context.Context, req SumRequest) (*SumResponse, error)
	Concat(ctx context.Context, req *ConcatRequest) (ConcatResponse, error)
}

type SumRequest struct {
	a, b int
	f    *model.Foo2 `pb:"seq=4,name=cc,type=int32"`
	c    int         `pb:"seq=4,name=cc,type=int32"`
}

type SumResponse struct {
	Code    int
	Message string
	Data    int
}

type ConcatRequest struct {
	a, b string
	F    Foo
}

type ConcatResponse struct {
	Code    int
	Message string
	Data    string
}

type Foo struct {
	Name       string
	Age        int
	F32        float32
	F64        float64
	UI32       uint32
	UI64       uint64
	B          bool
	StarInt    *int
	StarFoo    *Foo
	StarFoo2   *model.Foo2
	Bytes      []byte
	Strs       []string
	Ints       []int
	Int64s     []int64
	F32s       []float32
	F64s       []float64
	Foos       []Foo
	StarFoos   []*Foo
	Foos2      []model.Foo2
	StarFoos2  []*model.Foo2
	MS2I       map[string]int
	MS2I32     map[string]int32
	MS2I64     map[string]int64
	MS2UI32    map[string]uint32
	MS2UI64    map[string]uint64
	MI2I       map[int]int
	MS2ST      map[string]Foo
	MS2ST2     map[string]model.Foo2
	MS2StarST  map[string]*Foo
	MS2StarST2 map[string]*model.Foo2
}

// New returns a basic Service with all of the expected middlewares wired in.
func New(logger log.Logger, ints, chars metrics.Counter) Service {
	var svc Service
	{
		svc = NewBasicService()
		//svc = LoggingMiddleware(logger)(svc)
		//svc = InstrumentingMiddleware(ints, chars)(svc)
	}
	return svc
}

var (
	// ErrTwoZeroes is an arbitrary business rule for the Add method.
	ErrTwoZeroes = errors.New("can't sum two zeroes")

	// ErrIntOverflow protects the Add method. We've decided that this error
	// indicates a misbehaving service and should count against e.g. circuit
	// breakers. So, we return it directly in endpoints, to illustrate the
	// difference. In a real service, this probably wouldn't be the case.
	ErrIntOverflow = errors.New("integer overflow")

	// ErrMaxSizeExceeded protects the Concat method.
	ErrMaxSizeExceeded = errors.New("result exceeds maximum size")
)

// NewBasicService returns a naïve, stateless implementation of Service.
func NewBasicService() Service {
	return &basicService{}
}

type basicService struct{}

const (
	intMax = 1<<31 - 1
	intMin = -(intMax + 1)
	maxLen = 10
)

func (s *basicService) Sum(_ context.Context, req SumRequest) (*SumResponse, error) {
	a, b := req.a, req.b
	if a == 0 && b == 0 {
		return &SumResponse{Data: 0}, ErrTwoZeroes
	}
	if (b > 0 && a > (intMax-b)) || (b < 0 && a < (intMin-b)) {
		return &SumResponse{Data: 0}, ErrIntOverflow
	}
	return &SumResponse{Data: a + b}, nil
}

// Concat implements Service.
func (s basicService) Concat(_ context.Context, req *ConcatRequest) (ConcatResponse, error) {
	a, b := req.a, req.b
	if len(a)+len(b) > maxLen {
		return ConcatResponse{Data: ""}, ErrMaxSizeExceeded
	}
	return ConcatResponse{Data: fmt.Sprint(a + b)}, nil
}
