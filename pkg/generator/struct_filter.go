package generator

import (
	"strings"

	"ezrpro.com/micro/kit/pkg/cst"
)

type StructFilter func(s *cst.Struct) bool

func NoopStructFilter(s *cst.Struct) bool {
	return false
}

func DefaultStructFilter(s *cst.Struct) bool {
	structName := s.Name
	if !strings.HasSuffix(structName, "Request") &&
		!strings.HasSuffix(structName, "Response") {
		return true
	}

	return false
}
